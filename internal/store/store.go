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

	conn redis.Conn // for now using a single connection with a lock
	lock sync.Mutex
)

// prefix all keys with "kar\appName\"
func mangle(key string) string {
	return "kar" + config.Separator + config.AppName + config.Separator + key
}

// Set sets the value associated with a key
func Set(key, value string) (string, error) {
	lock.Lock()
	defer lock.Unlock()
	return redis.String(conn.Do("SET", mangle(key), value))
}

// SetNX sets the value associated with a key if it does not exist yet
func SetNX(key, value string) (int, error) {
	lock.Lock()
	defer lock.Unlock()
	return redis.Int(conn.Do("SETNX", mangle(key), value))
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
