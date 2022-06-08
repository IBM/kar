//
// Copyright IBM Corporation 2020,2022
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

// Package store provides an API for connecting to Redis
package store

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/IBM/kar/core/pkg/logger"
	"github.com/cenkalti/backoff/v4"
	"github.com/gomodule/redigo/redis"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// ErrNil indicates that a reply value is nil.
	ErrNil = redis.ErrNil

	// connection pool
	pool *redis.Pool

	// store configuration
	sc *StoreConfig

	requestDurationHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "kar_redis_request_durations_histogram_seconds",
		Help:    "KAR Redis request duration distributions.",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 12),
	})
)

type StoreConfig struct {
	// MangleKey is a hook to allow keys to be name mangled before they are passed through to Redis
	MangleKey func(string) string

	// UnmangleKey computes the inverse of MangleKey
	UnmangleKey func(string) string

	// RequestRetryLimit is how long to retry failing connections in a Redis call before giving up
	// A negative time will apply default durations
	RequestRetryLimit time.Duration

	// LongOperation sets a threshold used to report long-running redis operations
	LongOperation time.Duration

	// Host is the host of the Redis instance
	Host string

	// Port is the port of the Redis instance
	Port int

	// EnableTLS is set if the Redis connection requires TLS
	EnableTLS bool

	// RedisTLSSkipVerify is set to skip server name verification for Redis when connecting over TLS
	TLSSkipVerify bool

	// RedisPassword the password to use to connect to the Redis instance (required)
	Password string

	// RedisUser the user to use to connect to the Redis instance (required)
	User string

	// Redis certificate
	CA *x509.Certificate
}

func init() {
	prometheus.MustRegister(requestDurationHistogram)
}

func getValidConnection(ctx context.Context, limit time.Duration) (redis.Conn, error) {
	conn, err := pool.GetContext(ctx)
	if err == context.Canceled {
		return nil, err
	}
	if err != nil {
		b := backoff.NewExponentialBackOff()
		if limit >= 0 {
			b.MaxElapsedTime = limit
		}
		err = backoff.Retry(func() error {
			conn.Close()
			conn, err = pool.GetContext(ctx)
			if err == ctx.Err() {
				return backoff.Permanent(err)
			}
			return err
		}, b)
	}
	return conn, err
}

// send a command using a connection from the pool
func doRaw(ctx context.Context, command string, args ...interface{}) (reply interface{}, err error) {
	opStart := time.Now()
	conn, err := getValidConnection(ctx, sc.RequestRetryLimit)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	start := time.Now()
	reply, err = conn.Do(command, args...)
	if err != nil {
		panic(fmt.Sprintf("Failed to send command %v to redis: %v", command, err))
	}
	last := time.Now()
	elapsed := last.Sub(opStart)
	connElapsed := last.Sub(start)
	requestDurationHistogram.Observe(connElapsed.Seconds())
	if elapsed > sc.LongOperation {
		logger.Error("Slow Redis operation: %v total seconds (%v in conn.Do). Command was %v %v", elapsed.Seconds(), connElapsed.Seconds(), command, args[0])
	}
	return
}

// mangle the key before sending the command (assuming args[0] is the key)
func do(ctx context.Context, command string, args ...interface{}) (interface{}, error) {
	args[0] = sc.MangleKey(args[0].(string))
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
		return redis.Int(doRaw(ctx, "EVAL", "if redis.call('GET', KEYS[1]) == ARGV[1] then redis.call('DEL', KEYS[1]); return 1 else return 0 end", 1, sc.MangleKey(key), *expected))
	}
	return redis.Int(doRaw(ctx, "EVAL", "if redis.call('GET', KEYS[1]) == ARGV[1] then redis.call('SET', KEYS[1], ARGV[2]); return 1 else return 0 end", 1, sc.MangleKey(key), *expected, *value))
}

// Keys returns all keys that match the argument pattern
func Keys(ctx context.Context, pattern string) ([]string, error) {
	mangledKeys, err := redis.Strings(do(ctx, "KEYS", pattern))
	if err == nil {
		for idx, val := range mangledKeys {
			mangledKeys[idx] = sc.UnmangleKey(val)
		}
	}
	return mangledKeys, err
}

// Purge deletes all keys that match the argument pattern
func Purge(ctx context.Context, pattern string) (int, error) {
	pattern = sc.MangleKey(pattern)
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
func Dial(ctx context.Context, conf *StoreConfig) error {
	sc = conf // persist conf internally

	redisOptions := []redis.DialOption{}

	if conf.EnableTLS {
		redisOptions = append(redisOptions, redis.DialUseTLS(true))
		if sc.CA != nil {
			roots := x509.NewCertPool()
			roots.AddCert(sc.CA)
			redisOptions = append(redisOptions, redis.DialTLSConfig(&tls.Config{RootCAs: roots}))
		}
		if sc.TLSSkipVerify {
			redisOptions = append(redisOptions, redis.DialTLSSkipVerify(true))
		}
	}
	if sc.User != "" {
		redisOptions = append(redisOptions, redis.DialUsername(sc.User))
	}
	if sc.Password != "" {
		redisOptions = append(redisOptions, redis.DialPassword(sc.Password))
	}
	if sc.RequestRetryLimit >= 0 {
		redisOptions = append(redisOptions, redis.DialConnectTimeout(sc.RequestRetryLimit))
		redisOptions = append(redisOptions, redis.DialReadTimeout(sc.RequestRetryLimit))
		redisOptions = append(redisOptions, redis.DialWriteTimeout(sc.RequestRetryLimit))
	}

	address := net.JoinHostPort(sc.Host, strconv.Itoa(sc.Port))

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
	var limit time.Duration = sc.RequestRetryLimit
	if limit <= 0 {
		limit = 30 * time.Second
	}
	conn, err := getValidConnection(ctx, limit)
	if err == nil {
		defer conn.Close()
		_, err = conn.Do("PING")
	}
	return err
}

// Close terminates the connection pool.
func Close() error {
	return pool.Close()
}

// CAS sets the key to the desired value if the key has the expected value (expected != "") or is absent (expected == "")
// Returns the final value (original value if unchanged or desired if set)
func CAS(ctx context.Context, key string, expected string, desired string) (string, error) {
	script := "local v=redis.call('GET', KEYS[1]); if v==ARGV[1] or v==false and ARGV[1]=='' then redis.call('SET', KEYS[1], ARGV[2]); return ARGV[2] else return v end"
	value, err := redis.String(doRaw(ctx, "EVAL", script, 1, sc.MangleKey(key), expected, desired))
	if err == ErrNil {
		err = nil
	}
	return value, err
}
