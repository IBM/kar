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
	"crypto/tls"
	"crypto/x509"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/kar.git/core/internal/config"
	"github.com/IBM/kar.git/core/pkg/logger"
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

// send a command while holding the connection mutex
func doRaw(command string, args ...interface{}) (reply interface{}, err error) {
	opStart := time.Now()
	conn := pool.Get()
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
		logger.Error("failed to send command to redis: %v", err)
	}
	return
}

// mangle the key before sending the command (assuming args[0] is the key)
func do(command string, args ...interface{}) (interface{}, error) {
	args[0] = mangle(args[0].(string))
	return doRaw(command, args...)
}

// Keys

// Set sets the value associated with a key.
func Set(key, value string) (string, error) {
	return redis.String(do("SET", key, value))
}

// Get returns the value associated with a key.
func Get(key string) (string, error) {
	return redis.String(do("GET", key))
}

// Del deletes the value associated with a key.
func Del(key string) (int, error) {
	return redis.Int(do("DEL", key))
}

// CompareAndSet sets the value associated with a key if its current value is
// equal to the expected value. Use nil values to create or delete the key.
// Returns 0 if unsuccessful, 1 if successful.
func CompareAndSet(key string, expected, value *string) (int, error) {
	if expected == nil && value == nil {
		_, err := redis.String(do("GET", key))
		if err != nil {
			if err == ErrNil {
				return 1, nil
			}
			return 0, err
		}
		return 0, nil
	}
	if expected == nil {
		return redis.Int(do("SETNX", key, *value))
	}
	if value == nil {
		return redis.Int(doRaw("EVAL", "if redis.call('GET', KEYS[1]) == ARGV[1] then redis.call('DEL', KEYS[1]); return 1 else return 0 end", 1, mangle(key), *expected))
	}
	return redis.Int(doRaw("EVAL", "if redis.call('GET', KEYS[1]) == ARGV[1] then redis.call('SET', KEYS[1], ARGV[2]); return 1 else return 0 end", 1, mangle(key), *expected, *value))
}

// Keys returns all keys that match the argument pattern
func Keys(pattern string) ([]string, error) {
	mangledKeys, err := redis.Strings(do("KEYS", pattern))
	if err == nil {
		for idx, val := range mangledKeys {
			mangledKeys[idx] = unmangle(val)
		}
	}
	return mangledKeys, err
}

// Purge deletes all keys that match the argument pattern
func Purge(pattern string) (int, error) {
	pattern = mangle(pattern)
	bags := [][]interface{}{}
	cursor := 0
	for {
		reply, err := redis.Values(doRaw("SCAN", cursor, "MATCH", pattern, "COUNT", 100))
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
		n, err := redis.Int(doRaw("DEL", keys...))
		count += n
		if err != nil {
			return count, err
		}
	}
	return count, nil
}

// Hashes

// HSet hash key value
func HSet(hash, key, value string) (int, error) {
	return redis.Int(do("HSET", hash, key, value))
}

// HSet2 hash key1 value1 key2 value2
func HSet2(hash, key1, value1, key2, value2 string) (int, error) {
	return redis.Int(do("HSET", hash, key1, value1, key2, value2))
}

// HSet3 hash key1 value1 key2 value2 key3 value3
func HSet3(hash, key1, value1, key2, value2, key3, value3 string) (int, error) {
	return redis.Int(do("HSET", hash, key1, value1, key2, value2, key3, value3))
}

// HSetMultiple hash map[string]string does an HSET of the entire map
func HSetMultiple(hash string, keyValuePairs map[string]string) (int, error) {
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
	return redis.Int(do("HSET", args...))
}

// HGet hash key
func HGet(hash, key string) (string, error) {
	return redis.String(do("HGET", hash, key))
}

// HDel hash key
func HDel(hash, key string) (int, error) {
	return redis.Int(do("HDEL", hash, key))
}

//HDelMultiple hash key[]
func HDelMultiple(hash string, keys []string) (int, error) {
	args := make([]interface{}, len(keys)+1)
	args[0] = hash
	for i := range keys {
		args[i+1] = keys[i]
	}
	return redis.Int(do("HDEL", args...))
}

// HMGet hash key[]
func HMGet(hash string, keys []string) ([]string, error) {
	args := make([]interface{}, len(keys)+1)
	args[0] = hash
	for i := range keys {
		args[i+1] = keys[i]
	}
	return redis.Strings(do("HMGET", args...))
}

// HScan hash cursor [MATCH match]
func HScan(hash string, cursor int, match string) (int, []string, error) {
	var response []interface{}
	var err error
	if match != "" {
		response, err = redis.Values(do("HSCAN", hash, cursor, "MATCH", match, "COUNT", 1000)) // if we are filtering, increase count by quite a bit to compensate
	} else {
		response, err = redis.Values(do("HSCAN", hash, cursor))
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
func HGetAll(hash string) (map[string]string, error) {
	return redis.StringMap(do("HGETALL", hash))
}

// HExists hash key
func HExists(hash string, key string) (int, error) {
	return redis.Int(do("HEXISTS", hash, key))
}

// HKeys hash key
func HKeys(hash string) ([]string, error) {
	return redis.Strings(do("HKEYS", hash))
}

// Sorted sets

// ZAdd adds an element to a sorted set.
func ZAdd(key string, score int64, value string) (int, error) {
	return redis.Int(do("ZADD", key, score, value))
}

// ZRange returns a range of elements from a sorted set.
func ZRange(key string, start, stop int) ([]string, error) {
	return redis.Strings(do("ZRANGE", key, start, stop))
}

// ZRemRangeByScore removes elements by scores from a sorted set.
func ZRemRangeByScore(key string, min, max int64) (int, error) {
	return redis.Int(do("ZREMRANGEBYSCORE", key, min, max))
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
