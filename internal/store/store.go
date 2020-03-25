package store

import (
	"net"
	"strconv"
	"sync"

	"github.com/gomodule/redigo/redis"
	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

var (
	// ErrNil indicates that a reply value is nil
	ErrNil = redis.ErrNil

	// connection
	conn redis.Conn      // for now use a single connection
	lock = &sync.Mutex{} // and a lock
)

// mangle add common prefix to all keys
func mangle(key string) string {
	return "kar" + config.Separator + config.AppName + config.Separator + key
}

// send a command while holding the connection lock
func doRaw(command string, args ...interface{}) (reply interface{}, err error) {
	lock.Lock()
	reply, err = conn.Do(command, args...)
	lock.Unlock()
	return
}

// mangle the key before sending the command (assuming args[0] is the key)
func do(command string, args ...interface{}) (interface{}, error) {
	args[0] = mangle(args[0].(string))
	return doRaw(command, args...)
}

// Set sets the value associated with a key
func Set(key, value string) (string, error) {
	return redis.String(do("SET", key, value))
}

// RPush adds elements to a list
func RPush(key string, value string) (int, error) {
	return redis.Int(do("RPUSH", key, value))
}

// LRange returns elements from a list
func LRange(key string, start, stop int) ([]string, error) {
	return redis.Strings(do("LRANGE", key, start, stop))
}

// LRem removes elements from a list
func LRem(key string, count int, value string) (int, error) {
	return redis.Int(do("LREM", key, count, value))
}

// ZAdd adds elements to a sorted set
func ZAdd(key string, score int64, value string) (int, error) {
	return redis.Int(do("ZADD", key, score, value))
}

// ZRange returns elements from a sorted set
func ZRange(key string, start, stop int) ([]string, error) {
	return redis.Strings(do("ZRANGE", key, start, stop))
}

// ZRemRangeByScore removes elements from a sorted set
func ZRemRangeByScore(key string, min, max int64) (int, error) {
	return redis.Int(do("ZREMRANGEBYSCORE", key, min, max))
}

// CompareAndSet sets the value associated with a key if its current value is equal to the expected value
func CompareAndSet(key, expected, value string) (int, error) {
	if expected == "" {
		return redis.Int(do("SETNX", key, value))
	}
	if value == "" {
		return redis.Int(doRaw("EVAL", "if redis.call('GET', KEYS[1]) == ARGV[1] then redis.call('DEL', KEYS[1]); return 1 else return 0 end", 1, mangle(key), expected))
	}
	return redis.Int(doRaw("EVAL", "if redis.call('GET', KEYS[1]) == ARGV[1] then redis.call('SET', KEYS[1], ARGV[2]); return 1 else return 0 end", 1, mangle(key), expected, value))
}

// Get returns the value associated with the key
func Get(key string) (string, error) {
	return redis.String(do("GET", key))
}

// Del deletes the value associated with a key
func Del(key string) (int, error) {
	return redis.Int(do("DEL", key))
}

// Dial establishes a connection to the store
func Dial() {
	redisOptions := []redis.DialOption{}
	if config.RedisEnableTLS {
		redisOptions = append(redisOptions, redis.DialUseTLS(true))
		redisOptions = append(redisOptions, redis.DialTLSSkipVerify(true)) // TODO
	}
	if config.RedisPassword != "" {
		redisOptions = append(redisOptions, redis.DialPassword(config.RedisPassword))
	}
	var err error
	conn, err = redis.Dial("tcp", net.JoinHostPort(config.RedisHost, strconv.Itoa(config.RedisPort)), redisOptions...)
	if err != nil {
		logger.Fatal("failed to connect to Redis: %v", err)
	}
}

// Close closes the connection to the store
func Close() {
	// forcefully closing the connection appears to correctly and immediately
	// terminate pending requests as well as prevent new commands to be sent to
	// redis so there is no need for synchronization here
	err := conn.Close()
	if err != nil {
		logger.Fatal("failed to close connection to Redis: %v", err)
	}
}
