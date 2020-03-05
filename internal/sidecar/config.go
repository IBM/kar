package sidecar

import (
	"io/ioutil"
	"path/filepath"

	"github.ibm.com/solsa/kar.git/pkg/logger"
)

// Config captures the sidecar configuration
type Config struct {
	RedisHost     string
	RedisPort     string
	RedisPassword string
	KafkaBrokers  string
	KafkaUsername string
	KafkaPassword string
}

// LoadConfig reads the KAR runtime config from configVolume
func LoadConfig(configVolume string) Config {
	config := Config{
		RedisHost:     loadString(configVolume, "redis_host", true),
		RedisPort:     loadString(configVolume, "redis_port", true),
		RedisPassword: loadString(configVolume, "redis_password", true),
		KafkaBrokers:  loadString(configVolume, "kafka_brokers", true),
		KafkaUsername: loadString(configVolume, "kafka_username", true),
		KafkaPassword: loadString(configVolume, "kafka_password", true),
	}

	return config
}

func loadString(path string, file string, required bool) string {
	value := ""
	bytes, err := ioutil.ReadFile(filepath.Join(path, file))
	if err != nil {
		if required {
			logger.Fatal("Unable to read %v from %v", file, path)
		}
	} else {
		value = string(bytes)
	}
	return value
}
