// Package config loads the runtime configuration
package config

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

// Separator character for store keys and topic names
const Separator = "_" // must not be a legal DNS name character

var (
	// AppName is the name of the application
	AppName string

	// ServiceName is the name of the service
	ServiceName string

	// ActorTypes are the actor types implemented by this service
	ActorTypes []string

	// ActorCollectorInterval is the interval at which unused actors are collected
	ActorCollectorInterval time.Duration

	// ActorReminderInterval is the interval at which reminders are processed
	ActorReminderInterval time.Duration

	// ActorReminderAcceptableDelay controls the threshold at which reminders are logged as being late
	ActorReminderAcceptableDelay time.Duration

	// AppPort is the HTTP port the application process will be listening on
	AppPort int

	// RuntimePort is the HTTP port the runtime will be listening on
	RuntimePort int

	// KubernetesMode is true when this process is running in a sidecar container in a Kubernetes Pod
	KubernetesMode bool

	// PartitionZeroIneligible when true, Partition 0 will not be assigned to this sidecar
	PartitionZeroIneligible bool

	// KafkaBrokers is an array of Kafka brokers
	KafkaBrokers []string

	// KafkaEnableTLS is set if the Kafka connection requires TLS
	KafkaEnableTLS bool

	// KafkaUsername is the username for SASL authentication (optional)
	KafkaUsername string

	// KafkaPassword is the password for SASL authentication (optional)
	KafkaPassword string

	// KafkaVersion is the expected Kafka version
	KafkaVersion string

	// RedisHost is the host of the Redis instance
	RedisHost string

	// RedisPort is the port of the Redis instance
	RedisPort int

	// RedisEnableTLS is set if the Redis connection requires TLS
	RedisEnableTLS bool

	// RedisPassword the the password of the Redis instance (optional)
	RedisPassword string

	// ID is the unique id of this sidecar instance
	ID = uuid.New().String()

	// H2C enables h2c to communicate with the app service
	H2C bool

	// Purge the application state
	Purge bool
)

func init() {
	var kafkaBrokers, verbosity, configDir, actorTypes, collectInterval, remindInterval, remindDelay string
	var err error

	flag.StringVar(&AppName, "app", "", "The name of the application")
	flag.StringVar(&ServiceName, "service", "", "The name of the service provided by this process")
	flag.StringVar(&actorTypes, "actors", "", "The actor types provided by this process, as a comma separated list")
	flag.StringVar(&collectInterval, "actor_collector_interval", "10s", "Actor collector interval")
	flag.StringVar(&remindInterval, "actor_reminder_interval", "100ms", "Actor reminder processing interval")
	flag.StringVar(&remindDelay, "actor_reminder_acceptable_delay", "3s", "Threshold at which reminders are logged as being late")
	flag.IntVar(&AppPort, "app_port", 8080, "The port used by KAR to connect to the application")
	flag.IntVar(&RuntimePort, "runtime_port", 0, "The port used by the application to connect to KAR")
	flag.BoolVar(&KubernetesMode, "kubernetes_mode", false, "Running as a sidecar container in a Kubernetes Pod")
	flag.BoolVar(&PartitionZeroIneligible, "partition_zero_ineligible", false, "Is this sidecar ineligible to host partition zero?")
	flag.StringVar(&kafkaBrokers, "kafka_brokers", "", "The Kafka brokers to connect to, as a comma separated list")
	flag.BoolVar(&KafkaEnableTLS, "kafka_enable_tls", false, "Use TLS to communicate with Kafka")
	flag.StringVar(&KafkaUsername, "kafka_username", "", "The SASL username if any")
	flag.StringVar(&KafkaPassword, "kafka_password", "", "The SASL password if any")
	flag.StringVar(&KafkaVersion, "kafka_version", "", "Kafka cluster version")
	flag.StringVar(&RedisHost, "redis_host", "", "The Redis host")
	flag.IntVar(&RedisPort, "redis_port", 0, "The Redis port")
	flag.BoolVar(&RedisEnableTLS, "redis_enable_tls", false, "Use TLS to communicate with Redis")
	flag.StringVar(&RedisPassword, "redis_password", "", "The password of the Redis server if any")
	flag.StringVar(&verbosity, "v", "error", "Logging verbosity")
	flag.StringVar(&configDir, "config_dir", "", "Directory containing configuration files")
	flag.BoolVar(&H2C, "h2c", false, "Use h2c to communicate with service")
	flag.BoolVar(&Purge, "purge", false, "Purge the application state and exit")

	flag.Parse()

	logger.SetVerbosity(verbosity)

	if AppName == "" {
		logger.Fatal("app name is required")
	}

	if ServiceName == "" {
		ServiceName = "kar.none"
	}

	if actorTypes == "" {
		ActorTypes = []string{}
	} else {
		ActorTypes = strings.Split(actorTypes, ",")
	}

	ActorCollectorInterval, err = time.ParseDuration(collectInterval)
	if err != nil {
		logger.Fatal("error parsing actor_collector_interval %s", collectInterval)
	}

	ActorReminderInterval, err = time.ParseDuration(remindInterval)
	if err != nil {
		logger.Fatal("error parsing actor_reminder_interval %s", remindInterval)
	}

	ActorReminderAcceptableDelay, err = time.ParseDuration(remindDelay)
	if err != nil {
		logger.Fatal("error parsing actor_reminder_acceptable_delay %s", remindDelay)
	}

	if !KafkaEnableTLS && os.Getenv("KAFKA_ENABLE_TLS") != "" {
		if KafkaEnableTLS, err = strconv.ParseBool(os.Getenv("KAFKA_ENABLE_TLS")); err != nil {
			logger.Fatal("error parsing environment variable KAFKA_ENABLE_TLS")
		}
	}

	if kafkaBrokers == "" {
		if kafkaBrokers = os.Getenv("KAFKA_BROKERS"); kafkaBrokers == "" {
			if kafkaBrokers = loadStringFromConfig(configDir, "kafka_brokers"); kafkaBrokers == "" {
				logger.Fatal("at least one Kafka broker is required")
			}
		}
	}

	KafkaBrokers = strings.Split(kafkaBrokers, ",")

	if KafkaUsername == "" {
		if KafkaUsername = os.Getenv("KAFKA_USERNAME"); KafkaUsername == "" {
			if KafkaUsername = loadStringFromConfig(configDir, "kafka_username"); KafkaUsername == "" {
				KafkaUsername = "token"
			}
		}
	}

	if KafkaPassword == "" {
		if KafkaPassword = os.Getenv("KAFKA_PASSWORD"); KafkaPassword == "" {
			KafkaPassword = loadStringFromConfig(configDir, "kafka_password")
		}
	}

	if KafkaVersion == "" {
		if KafkaVersion = os.Getenv("KAFKA_VERSION"); KafkaVersion == "" {
			if KafkaVersion = loadStringFromConfig(configDir, "kafka_version"); KafkaVersion == "" {
				KafkaVersion = "2.2.0"
			}
		}
	}

	if !RedisEnableTLS && os.Getenv("REDIS_ENABLE_TLS") != "" {
		if RedisEnableTLS, err = strconv.ParseBool(os.Getenv("REDIS_ENABLE_TLS")); err != nil {
			logger.Fatal("error parsing environment variable REDIS_ENABLE_TLS")
		}
	}

	if RedisHost == "" {
		if RedisHost = os.Getenv("REDIS_HOST"); RedisHost == "" {
			if RedisHost = loadStringFromConfig(configDir, "redis_host"); RedisHost == "" {
				logger.Fatal("Redis host is required")
			}
		}
	}

	if RedisPort == 0 {
		if os.Getenv("REDIS_PORT") != "" {
			if RedisPort, err = strconv.Atoi(os.Getenv("REDIS_PORT")); err != nil {
				logger.Fatal("error parsing environment variable REDIS_PORT")
			}
		} else {
			if rp := loadStringFromConfig(configDir, "redis_port"); rp != "" {
				if RedisPort, err = strconv.Atoi(rp); err != nil {
					logger.Fatal("error parsing config value for redis_port: %s", rp)
				}
			} else {
				RedisPort = 6379
			}
		}
	}

	if RedisPassword == "" {
		if RedisPassword = os.Getenv("REDIS_PASSWORD"); RedisPassword == "" {
			RedisPassword = loadStringFromConfig(configDir, "redis_password")
		}
	}
}

func loadStringFromConfig(path string, file string) string {
	value := ""
	if path != "" {
		if bytes, err := ioutil.ReadFile(filepath.Join(path, file)); err == nil {
			value = string(bytes)
		}
	}
	return value
}
