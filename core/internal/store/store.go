//
// Copyright IBM Corporation 2020,2021
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

// Package store implements the store API.
package store

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/kar/core/internal/config"
	"github.com/IBM/kar/core/pkg/logger"
	"github.com/cenkalti/backoff/v4"
	"github.com/gomodule/redigo/redis"
)

var (
	// ErrNil indicates that a reply value is nil.
	ErrNil = redis.ErrNil

	// connection pool
	pool *redis.Pool
)

// mangle add common prefix to all keys
func mangle(key string) string {
	return "kar" + config.Separator + config.AppName + config.Separator + key
}

// unmangle a key by removing the common prefix if it has it
func unmangle(key string) string {
	parts := strings.Split(key, config.Separator)
	if parts[0] == "kar" && parts[1] == config.AppName {
		return strings.Join(parts[2:], config.Separator)
	}
	return key
}

// send a command using a connection from the pool
func doRaw(ctx context.Context, command string, args ...interface{}) (reply interface{}, err error) {
	opStart := time.Now()
	conn, err := pool.GetContext(ctx)
	if err != nil {
		b := backoff.NewExponentialBackOff()
		if config.RequestRetryLimit >= 0 {
			b.MaxElapsedTime = config.RequestRetryLimit
		}
		err = backoff.Retry(func() error {
			conn.Close()
			conn, err = pool.GetContext(ctx)
			return err
		}, b)
	}
	defer conn.Close()
	start := time.Now()
	reply, err = conn.Do(command, args...)
	last := time.Now()
	elapsed := last.Sub(opStart)
	connElapsed := last.Sub(start)
	if elapsed > config.LongRedisOperation {
		logger.Error("Slow Redis operation: %v total seconds (%v in conn.Do). Command was %v %v", elapsed.Seconds(), connElapsed.Seconds(), command, args[0])
	}
	if err != nil {
		logger.Fatal("Failed to send command %v to redis: %v", command, err)
	}
	return
}

// mangle the key before sending the command (assuming args[0] is the key)
func do(ctx context.Context, command string, args ...interface{}) (interface{}, error) {
	args[0] = mangle(args[0].(string))
	return doRaw(ctx, command, args...)
}

// Keys

// Set sets the value associated with a key.
func Set(ctx context.Context, key, value string) (string, error) {
	return redis.String(do(ctx, "SET", key, value))
}

// Get returns the value associated with a key.
func Get(ctx context.Context, key string) (string, error) {
	return redis.String(do(ctx, "GET", key))
}

// Del deletes the value associated with a key.
func Del(ctx context.Context, key string) (int, error) {
	return redis.Int(do(ctx, "DEL", key))
}

// CompareAndSet sets the value associated with a key if its current value is
// equal to the expected value. Use nil values to create or delete the key.
// Returns 0 if unsuccessful, 1 if successful.
func CompareAndSet(ctx context.Context, key string, expected, value *string) (int, error) {
	if expected == nil && value == nil {
		_, err := redis.String(do(ctx, "GET", key))
		if err != nil {
			if err == ErrNil {
				return 1, nil
			}
			return 0, err
		}
		return 0, nil
	}
	if expected == nil {
		return redis.Int(do(ctx, "SETNX", key, *value))
	}
	if value == nil {
		return redis.Int(doRaw(ctx, "EVAL", "if redis.call('GET', KEYS[1]) == ARGV[1] then redis.call('DEL', KEYS[1]); return 1 else return 0 end", 1, mangle(key), *expected))
	}
	return redis.Int(doRaw(ctx, "EVAL", "if redis.call('GET', KEYS[1]) == ARGV[1] then redis.call('SET', KEYS[1], ARGV[2]); return 1 else return 0 end", 1, mangle(key), *expected, *value))
}

// Keys returns all keys that match the argument pattern
func Keys(ctx context.Context, pattern string) ([]string, error) {
	mangledKeys, err := redis.Strings(do(ctx, "KEYS", pattern))
	if err == nil {
		for idx, val := range mangledKeys {
			mangledKeys[idx] = unmangle(val)
		}
	}
	return mangledKeys, err
}

// Purge deletes all keys that match the argument pattern
func Purge(ctx context.Context, pattern string) (int, error) {
	pattern = mangle(pattern)
	bags := [][]interface{}{}
	cursor := 0
	for {
		reply, err := redis.Values(doRaw(ctx, "SCAN", cursor, "MATCH", pattern, "COUNT", 100))
		if err != nil {
			return 0, err
		}
		cursor, _ = strconv.Atoi(string(reply[0].([]byte)))
		keys := reply[1].([]interface{})
		if len(keys) > 0 {
			bags = append(bags, keys)
		}
		if cursor == 0 {
			break
		}
	}
	count := 0
	for _, keys := range bags {
		n, err := redis.Int(doRaw(ctx, "DEL", keys...))
		count += n
		if err != nil {
			return count, err
		}
	}
	return count, nil
}

// Hashes

// HSet hash key value
func HSet(ctx context.Context, hash, key, value string) (int, error) {
	return redis.Int(do(ctx, "HSET", hash, key, value))
}

// HSet2 hash key1 value1 key2 value2
func HSet2(ctx context.Context, hash, key1, value1, key2, value2 string) (int, error) {
	return redis.Int(do(ctx, "HSET", hash, key1, value1, key2, value2))
}

// HSet3 hash key1 value1 key2 value2 key3 value3
func HSet3(ctx context.Context, hash, key1, value1, key2, value2, key3, value3 string) (int, error) {
	return redis.Int(do(ctx, "HSET", hash, key1, value1, key2, value2, key3, value3))
}

// HSetMultiple hash map[string]string does an HSET of the entire map
func HSetMultiple(ctx context.Context, hash string, keyValuePairs map[string]string) (int, error) {
	nPairs := len(keyValuePairs)
	if nPairs == 0 {
		return 0, nil
	}
	args := make([]interface{}, 2*nPairs+1)
	args[0] = hash
	idx := 1
	for k, v := range keyValuePairs {
		args[idx] = k
		args[idx+1] = v
		idx += 2
	}
	return redis.Int(do(ctx, "HSET", args...))
}

// HGet hash key
func HGet(ctx context.Context, hash, key string) (string, error) {
	return redis.String(do(ctx, "HGET", hash, key))
}

// HDel hash key
func HDel(ctx context.Context, hash, key string) (int, error) {
	return redis.Int(do(ctx, "HDEL", hash, key))
}

//HDelMultiple hash key[]
func HDelMultiple(ctx context.Context, hash string, keys []string) (int, error) {
	args := make([]interface{}, len(keys)+1)
	args[0] = hash
	for i := range keys {
		args[i+1] = keys[i]
	}
	return redis.Int(do(ctx, "HDEL", args...))
}

// HMGet hash key[]
func HMGet(ctx context.Context, hash string, keys []string) ([]string, error) {
	args := make([]interface{}, len(keys)+1)
	args[0] = hash
	for i := range keys {
		args[i+1] = keys[i]
	}
	return redis.Strings(do(ctx, "HMGET", args...))
}

// HScan hash cursor [MATCH match]
func HScan(ctx context.Context, hash string, cursor int, match string) (int, []string, error) {
	var response []interface{}
	var err error
	if match != "" {
		response, err = redis.Values(do(ctx, "HSCAN", hash, cursor, "MATCH", match, "COUNT", 1000)) // if we are filtering, increase count by quite a bit to compensate
	} else {
		response, err = redis.Values(do(ctx, "HSCAN", hash, cursor))
	}
	if err != nil {
		return 0, nil, err
	}
	cursor, err = strconv.Atoi(string(response[0].([]byte)))
	if err != nil {
		return 0, nil, err
	}
	data := response[1].([]interface{})
	ans := make([]string, len(data))
	for i := range data {
		ans[i] = string(data[i].([]byte))
	}
	return cursor, ans, nil
}

// HGetAll hash
func HGetAll(ctx context.Context, hash string) (map[string]string, error) {
	return redis.StringMap(do(ctx, "HGETALL", hash))
}

// HExists hash key
func HExists(ctx context.Context, hash string, key string) (int, error) {
	return redis.Int(do(ctx, "HEXISTS", hash, key))
}

// HKeys hash key
func HKeys(ctx context.Context, hash string) ([]string, error) {
	return redis.Strings(do(ctx, "HKEYS", hash))
}

// Sorted sets

// ZAdd adds an element to a sorted set.
func ZAdd(ctx context.Context, key string, score int64, value string) (int, error) {
	return redis.Int(do(ctx, "ZADD", key, score, value))
}

// ZRange returns a range of elements from a sorted set.
func ZRange(ctx context.Context, key string, start, stop int) ([]string, error) {
	return redis.Strings(do(ctx, "ZRANGE", key, start, stop))
}

// ZRemRangeByScore removes elements by scores from a sorted set.
func ZRemRangeByScore(ctx context.Context, key string, min, max int64) (int, error) {
	return redis.Int(do(ctx, "ZREMRANGEBYSCORE", key, min, max))
}

// Dial connects to Redis.
func Dial() error {
	redisOptions := []redis.DialOption{}

	if config.RedisEnableTLS {
		redisOptions = append(redisOptions, redis.DialUseTLS(true))
		if config.RedisCA != nil {
			roots := x509.NewCertPool()
			roots.AddCert(config.RedisCA)
			redisOptions = append(redisOptions, redis.DialTLSConfig(&tls.Config{RootCAs: roots}))
		}
		if config.RedisTLSSkipVerify {
			redisOptions = append(redisOptions, redis.DialTLSSkipVerify(true))
		}
	}
	if config.RedisUser != "" {
		redisOptions = append(redisOptions, redis.DialUsername(config.RedisUser))
	}
	if config.RedisPassword != "" {
		redisOptions = append(redisOptions, redis.DialPassword(config.RedisPassword))
	}
	if config.RequestRetryLimit >= 0 {
		redisOptions = append(redisOptions, redis.DialConnectTimeout(config.RequestRetryLimit))
		redisOptions = append(redisOptions, redis.DialReadTimeout(config.RequestRetryLimit))
		redisOptions = append(redisOptions, redis.DialWriteTimeout(config.RequestRetryLimit))
	}

	address := net.JoinHostPort(config.RedisHost, strconv.Itoa(config.RedisPort))

	pool = &redis.Pool{
		MaxIdle:     3,
		MaxActive:   16,
		IdleTimeout: 240 * time.Second,
		Wait:        true,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", address, redisOptions...)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
	conn := pool.Get()
	defer conn.Close()
	_, err := conn.Do("PING")
	return err
}

// Close terminates the connection pool.
func Close() error {
	return pool.Close()
}
