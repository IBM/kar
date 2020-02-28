// Package config loads the runtime configuration
package config

import (
	"flag"
	"os"
	"strconv"
	"strings"

	"github.com/Shopify/sarama"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

var (
	AppName string

	ServiceName string
	ServicePort int

	RuntimePort int

	KafkaBrokers  []string
	KafkaTLS      bool
	KafkaUser     string
	KafkaPassword string
	KafkaVersion  sarama.KafkaVersion

	RedisAddress  string
	RedisTLS      bool
	RedisPassword string
)

func init() {
	var kafkaBrokers string
	var kafkaVersion string
	var verbosity int
	var err error

	flag.StringVar(&AppName, "app", "", "The name of the application")
	flag.StringVar(&ServiceName, "service", "", "The name of the service being joined to the application")
	flag.IntVar(&ServicePort, "port", 3000, "The HTTP port for the service")
	flag.IntVar(&RuntimePort, "listen", 0, "The HTTP port for KAR to listen on") // defaults to 0 for dynamic selection
	flag.StringVar(&kafkaBrokers, "kafka_brokers", "", "The Kafka brokers to connect to, as a comma separated list")
	flag.BoolVar(&KafkaTLS, "kafka_tls", false, "Use TLS to communicate with Kafka")
	flag.StringVar(&KafkaUser, "kafka_user", "", "The SASL username if any")
	flag.StringVar(&KafkaPassword, "kafka_password", "", "The SASL password if any")
	flag.StringVar(&kafkaVersion, "kafka_version", "", "Kafka cluster version")
	flag.StringVar(&RedisAddress, "redis_address", "", "The address of the Redis server to connect to")
	flag.BoolVar(&RedisTLS, "redis_tls", false, "Use TLS to communicate with Redis")
	flag.StringVar(&RedisPassword, "redis_password", "", "The password of the Redis server if any")
	flag.IntVar(&verbosity, "v", 0, "Logging verbosity")

	flag.Parse()

	logger.SetVerbosity(logger.Severity(verbosity))

	if AppName == "" {
		logger.Fatal("app name is required")
	}

	if ServiceName == "" {
		logger.Fatal("service name is required")
	}

	if !KafkaTLS && os.Getenv("KAFKA_TLS") != "" {
		if KafkaTLS, err = strconv.ParseBool(os.Getenv("KAFKA_TLS")); err != nil {
			logger.Fatal("error parsing environment variable KAFKA_TLS")
		}
	}

	if kafkaBrokers == "" {
		if kafkaBrokers = os.Getenv("KAFKA_BROKERS"); kafkaBrokers == "" {
			logger.Fatal("at least one Kafka broker is required")
		}
	}

	KafkaBrokers = strings.Split(kafkaBrokers, ",")

	if KafkaUser == "" {
		if KafkaUser = os.Getenv("KAFKA_USER"); KafkaUser == "" {
			KafkaUser = "token"
		}
	}

	if KafkaPassword == "" {
		KafkaPassword = os.Getenv("KAFKA_PASSWORD")
	}

	if kafkaVersion == "" {
		if kafkaVersion = os.Getenv("KAFKA_VERSION"); kafkaVersion == "" {
			kafkaVersion = "2.2.0"
		}
	}

	if KafkaVersion, err = sarama.ParseKafkaVersion(kafkaVersion); err != nil {
		logger.Fatal("invalid Kafka version: %v", err)
	}

	if !RedisTLS && os.Getenv("REDIS_TLS") != "" {
		if RedisTLS, err = strconv.ParseBool(os.Getenv("REDIS_TLS")); err != nil {
			logger.Fatal("error parsing environment variable REDIS_TLS")
		}
	}

	if RedisAddress == "" {
		if RedisAddress = os.Getenv("REDIS_ADDRESS"); RedisAddress == "" {
			logger.Fatal("address of Redis is required")
		}
	}

	if RedisPassword == "" {
		RedisPassword = os.Getenv("REDIS_PASSWORD")
	}
}
