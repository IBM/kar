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
func Set(key, value string) (string, error) {
	lock.Lock()
	defer lock.Unlock()
	return redis.String(conn.Do("SET", mangle(key), value))
}

// CompareAndSet sets the value associated with a key if its current value is equal to the expected value
func CompareAndSet(key, expected, value string) (int, error) {
	lock.Lock()
	defer lock.Unlock()
	if expected == "" {
		return redis.Int(conn.Do("SETNX", mangle(key), value))
	}
	return redis.Int(conn.Do("EVAL", "if redis.call('GET', KEYS[1]) == ARGV[1] then redis.call('SET', KEYS[1], ARGV[2]); return 1 else return 0 end", 1, mangle(key), expected, value))
}

// Get returns the value associated with the key
func Get(key string) (string, error) {
	lock.Lock()
	defer lock.Unlock()
	return redis.String(conn.Do("GET", mangle(key)))
}

// Del deletes the value associated with a key
func Del(key string) (int, error) {
	lock.Lock()
	defer lock.Unlock()
	return redis.Int(conn.Do("DEL", mangle(key)))
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
