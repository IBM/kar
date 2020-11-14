// Package store implements the store API.
package store

import (
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.ibm.com/solsa/kar.git/core/internal/config"
	"github.ibm.com/solsa/kar.git/core/pkg/logger"
)

var (
	// ErrNil indicates that a reply value is nil.
	ErrNil = redis.ErrNil

	// connection
	conn redis.Conn      // for now use a single connection
	mu   = &sync.Mutex{} // and a mutex

	last = time.Now()
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
	mu.Lock()
	if time.Since(last) > time.Minute {
		// check connection and reconnect if necessary
		conn.Do("PING")
		err = conn.Err()
		if err != nil {
			err = Dial()
			if err != nil {
				logger.Error("failed to reconnect to redis: %v", err)
				return
			}
		}
	}
	reply, err = conn.Do(command, args...)
	last = time.Now()
	mu.Unlock()
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

// Connection

// Dial establishes a connection to the store.
func Dial() error {
	redisOptions := []redis.DialOption{}
	if config.RedisEnableTLS {
		redisOptions = append(redisOptions, redis.DialUseTLS(true))
		redisOptions = append(redisOptions, redis.DialTLSSkipVerify(true)) // TODO
	}
	if config.RedisPassword != "" {
		redisOptions = append(redisOptions, redis.DialPassword(config.RedisPassword))
	}
	if config.RequestTimeout >= 0 {
		redisOptions = append(redisOptions, redis.DialConnectTimeout(config.RequestTimeout))
		redisOptions = append(redisOptions, redis.DialReadTimeout(config.RequestTimeout))
		redisOptions = append(redisOptions, redis.DialWriteTimeout(config.RequestTimeout))
	}
	var err error
	conn, err = redis.Dial("tcp", net.JoinHostPort(config.RedisHost, strconv.Itoa(config.RedisPort)), redisOptions...)
	return err
}

// Close closes the connection to the store.
func Close() error {
	// forcefully closing the connection appears to correctly and immediately
	// terminate pending requests as well as prevent new commands to be sent to
	// redis so there is no need for synchronization here
	return conn.Close()
}
