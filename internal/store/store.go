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
	conn redis.Conn // for now using a single connection with a lock
	lock sync.Mutex
)

// Separator character
const Separator = "\\"

// prefix all keys with "kar\appName\"
func mangle(key string) string {
	return "kar" + Separator + config.AppName + Separator + key
}

// Set sets the value associated with a key
func Set(key, value string) error {
	lock.Lock()
	defer lock.Unlock()
	_, err := conn.Do("SET", mangle(key), value)
	return err
}

// Get returns the value associated with the key
func Get(key string) (*string, error) {
	lock.Lock()
	defer lock.Unlock()
	reply, err := conn.Do("GET", mangle(key))
	if reply == nil || err != nil {
		return nil, err
	}
	value := string(reply.([]byte))
	return &value, nil
}

// Del deletes the value associated with a key
func Del(key string) error {
	lock.Lock()
	defer lock.Unlock()
	_, err := conn.Do("DEL", mangle(key))
	return err
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
