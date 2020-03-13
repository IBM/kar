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
	conn redis.Conn // for now use a single connection
	lock sync.Mutex // connection lock
)

// mangle add common prefix to all keys
func mangle(key string) string {
	return "kar" + config.Separator + config.AppName + config.Separator + key
}

// Set sets the value associated with a key
func Set(key, value string) (reply string, err error) {
	lock.Lock()
	reply, err = redis.String(conn.Do("SET", mangle(key), value))
	lock.Unlock()
	return
}

// RPush adds elements to a list
func RPush(key string, value string) (reply int, err error) {
	lock.Lock()
	reply, err = redis.Int(conn.Do("RPUSH", mangle(key), value))
	lock.Unlock()
	return
}

// LRange returns elements from a list
func LRange(key string, start, stop int) (reply []string, err error) {
	lock.Lock()
	reply, err = redis.Strings(conn.Do("LRANGE", mangle(key), start, stop))
	lock.Unlock()
	return
}

// LRem removes elements from a list
func LRem(key string, count int, value string) (reply int, err error) {
	lock.Lock()
	reply, err = redis.Int(conn.Do("LREM", mangle(key), count, value))
	lock.Unlock()
	return
}

// ZAdd adds elements to a sorted set
func ZAdd(key string, score int64, value string) (reply int, err error) {
	lock.Lock()
	reply, err = redis.Int(conn.Do("ZADD", mangle(key), score, value))
	lock.Unlock()
	return
}

// ZRange returns elements from a sorted set
func ZRange(key string, start, stop int) (reply []string, err error) {
	lock.Lock()
	reply, err = redis.Strings(conn.Do("ZRANGE", mangle(key), start, stop))
	lock.Unlock()
	return
}

// ZRemRangeByScore removes elements from a sorted set
func ZRemRangeByScore(key string, min, max int64) (reply int, err error) {
	lock.Lock()
	reply, err = redis.Int(conn.Do("ZREMRANGEBYSCORE", mangle(key), min, max))
	lock.Unlock()
	return
}

// CompareAndSet sets the value associated with a key if its current value is equal to the expected value
func CompareAndSet(key, expected, value string) (reply int, err error) {
	lock.Lock()
	if expected == "" {
		reply, err = redis.Int(conn.Do("SETNX", mangle(key), value))
	} else {
		reply, err = redis.Int(conn.Do("EVAL", "if redis.call('GET', KEYS[1]) == ARGV[1] then redis.call('SET', KEYS[1], ARGV[2]); return 1 else return 0 end", 1, mangle(key), expected, value))
	}
	lock.Unlock()
	return
}

// Get returns the value associated with the key
func Get(key string) (reply string, err error) {
	lock.Lock()
	reply, err = redis.String(conn.Do("GET", mangle(key)))
	lock.Unlock()
	return
}

// Del deletes the value associated with a key
func Del(key string) (reply int, err error) {
	lock.Lock()
	reply, err = redis.Int(conn.Do("DEL", mangle(key)))
	lock.Unlock()
	return
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
	conn.Close()
}
