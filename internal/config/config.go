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
	// AppName is the name of the application
	AppName string

	// ServiceName is the name of the service
	ServiceName string

	// ServicePort is the HTTP port the service will be listening on
	ServicePort int

	// RuntimePort is the HTTP port the runtime will be listening on
	RuntimePort int

	// KafkaBrokers is an array of Kafka brokers
	KafkaBrokers []string

	// KafkaEnableTLS is set if the Kafka connection requires TLS
	KafkaEnableTLS bool

	// KafkaUsername is the username for SASL authentication (optional)
	KafkaUsername string

	// KafkaPassword is the password for SASL authentication (optional)
	KafkaPassword string

	// KafkaVersion is the expected Kafka version
	KafkaVersion sarama.KafkaVersion

	// RedisHost is the host of the Redis instance
	RedisHost string

	// RedisPort is the port of the Redis instance
	RedisPort int

	// RedisEnableTLS is set if the Redis connection requires TLS
	RedisEnableTLS bool

	// RedisPassword the the password of the Redis instance (optional)
	RedisPassword string
)

func init() {
	var kafkaBrokers, kafkaVersion string
	var verbosity int
	var err error

	flag.StringVar(&AppName, "app", "", "The name of the application")
	flag.StringVar(&ServiceName, "service", "", "The name of the service being joined to the application")
	flag.IntVar(&ServicePort, "send", 3000, "The service port")
	flag.IntVar(&RuntimePort, "recv", 0, "The runtime port")
	flag.StringVar(&kafkaBrokers, "kafka_brokers", "", "The Kafka brokers to connect to, as a comma separated list")
	flag.BoolVar(&KafkaEnableTLS, "kafka_enable_tls", false, "Use TLS to communicate with Kafka")
	flag.StringVar(&KafkaUsername, "kafka_username", "", "The SASL username if any")
	flag.StringVar(&KafkaPassword, "kafka_password", "", "The SASL password if any")
	flag.StringVar(&kafkaVersion, "kafka_version", "", "Kafka cluster version")
	flag.StringVar(&RedisHost, "redis_host", "", "The Redis host")
	flag.IntVar(&RedisPort, "redis_port", 0, "The Redis port")
	flag.BoolVar(&RedisEnableTLS, "redis_enable_tls", false, "Use TLS to communicate with Redis")
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

	if !KafkaEnableTLS && os.Getenv("KAFKA_ENABLE_TLS") != "" {
		if KafkaEnableTLS, err = strconv.ParseBool(os.Getenv("KAFKA_ENABLE_TLS")); err != nil {
			logger.Fatal("error parsing environment variable KAFKA_ENABLE_TLS")
		}
	}

	if kafkaBrokers == "" {
		if kafkaBrokers = os.Getenv("KAFKA_BROKERS"); kafkaBrokers == "" {
			logger.Fatal("at least one Kafka broker is required")
		}
	}

	KafkaBrokers = strings.Split(kafkaBrokers, ",")

	if KafkaUsername == "" {
		if KafkaUsername = os.Getenv("KAFKA_USERNAME"); KafkaUsername == "" {
			KafkaUsername = "token"
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

	if !RedisEnableTLS && os.Getenv("REDIS_ENABLE_TLS") != "" {
		if RedisEnableTLS, err = strconv.ParseBool(os.Getenv("REDIS_ENABLE_TLS")); err != nil {
			logger.Fatal("error parsing environment variable REDIS_ENABLE_TLS")
		}
	}

	if RedisHost == "" {
		if RedisHost = os.Getenv("REDIS_HOST"); RedisHost == "" {
			logger.Fatal("Redis host is required")
		}
	}

	if RedisPort == 0 {
		if os.Getenv("REDIS_PORT") != "" {
			if RedisPort, err = strconv.Atoi(os.Getenv("REDIS_PORT")); err != nil {
				logger.Fatal("error parsing environment variable REDIS_PORT")
			}
		} else {
			RedisPort = 6379
		}
	}

	if RedisPassword == "" {
		RedisPassword = os.Getenv("REDIS_PASSWORD")
	}
}
