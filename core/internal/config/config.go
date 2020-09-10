// Package config loads the runtime configuration
package config

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.ibm.com/solsa/kar.git/core/pkg/logger"
)

// Separator character for store keys and topic names
const Separator = "_" // must not be a legal DNS name character

var (
	// CmdName is the name of the kar command
	CmdName string

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

	// Purge the application state and messages
	Purge bool

	// Drain the application messages
	Drain bool

	// Invoke an actor method
	Invoke bool

	// Hostname is the name of the host
	Hostname string

	// RequestTimeout is how long to wait in a Redis/http call before timing out
	// A negative time will apply default durations
	RequestTimeout time.Duration

	// ActorTimeout is how long to wait on a busy actor before timing it out and returning the error.
	ActorTimeout time.Duration

	// OutputStyle is whether to print a human readable output, or return a JSON string of data.
	// Currently only applies to calling system/information/
	OutputStyle string

	// Outputs information on running systems, like calling system/information/<Get>
	// Currently supported: Sidecars
	Get string

	// temporary variables to parse command line options
	kafkaBrokers, verbosity, configDir, actorTypes, collectInterval, remindInterval, remindDelay, timeoutTime, actorTimeoutTime string
)

// define the flags available on all commands
func globalOptions(f *flag.FlagSet) {
	f.StringVar(&AppName, "app", "", "The name of the application (required)")

	f.StringVar(&kafkaBrokers, "kafka_brokers", "", "The Kafka brokers to connect to, as a comma separated list")
	f.BoolVar(&KafkaEnableTLS, "kafka_enable_tls", false, "Use TLS to communicate with Kafka")
	f.StringVar(&KafkaUsername, "kafka_username", "", "The SASL username if any")
	f.StringVar(&KafkaPassword, "kafka_password", "", "The SASL password if any")
	f.StringVar(&KafkaVersion, "kafka_version", "", "Kafka cluster version")

	f.StringVar(&RedisHost, "redis_host", "", "The Redis host")
	f.IntVar(&RedisPort, "redis_port", 0, "The Redis port")
	f.BoolVar(&RedisEnableTLS, "redis_enable_tls", false, "Use TLS to communicate with Redis")
	f.StringVar(&RedisPassword, "redis_password", "", "The password of the Redis server if any")

	f.StringVar(&timeoutTime, "timeout", "-1s", "Time to wait before timing out calls")

	f.StringVar(&verbosity, "v", "error", "Logging verbosity")

	f.StringVar(&configDir, "config_dir", "", "Directory containing configuration files")
}

func init() {
	var err error

	usage := `kar COMMAND ...

Available commands:
  run     run application component
  get     query running application
  invoke  invoke actor instance
  purge   purge application messages and state
  drain   drain application messages
  help    print help message`

	description := `Use "kar COMMAND -h" for more information about a command`

	options := ""

	writer := os.Stderr

	// print usage to writer
	flag.Usage = func() {
		// print command usage and description
		fmt.Fprintf(writer, "Usage:\n  %s\n\n%s\n\n", usage, description)

		// print command-specific options if any
		if options != "" {
			fmt.Fprintf(writer, "Options:\n%s\n", options)
		}

		// print global options by creating a dummy flag set
		fmt.Fprintln(writer, "Global Options:")
		f := flag.NewFlagSet("", flag.ContinueOnError)
		globalOptions(f)
		f.SetOutput(writer)
		f.PrintDefaults()
	}

	// missing command
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "invalid command")
		flag.Usage()
		os.Exit(2)
	}

	CmdName := os.Args[1]

	switch CmdName {
	case "run":
		usage = "kar run [OPTIONS] [-- [COMMAND]]"
		description = "Run application component"
		flag.StringVar(&ServiceName, "service", "", "The name of the service provided by this process")
		flag.StringVar(&actorTypes, "actors", "", "The actor types provided by this process, as a comma separated list")
		flag.StringVar(&collectInterval, "actor_collector_interval", "10s", "Actor collector interval")
		flag.StringVar(&remindInterval, "actor_reminder_interval", "100ms", "Actor reminder processing interval")
		flag.StringVar(&remindDelay, "actor_reminder_acceptable_delay", "3s", "Threshold at which reminders are logged as being late")
		flag.IntVar(&AppPort, "app_port", 8080, "The port used by KAR to connect to the application")
		flag.IntVar(&RuntimePort, "runtime_port", 0, "The port used by the application to connect to KAR")
		flag.BoolVar(&KubernetesMode, "kubernetes_mode", false, "Running as a sidecar container in a Kubernetes Pod")
		flag.BoolVar(&H2C, "h2c", false, "Use h2c to communicate with service")
		flag.StringVar(&Hostname, "hostname", "localhost", "Hostname")
		flag.StringVar(&actorTimeoutTime, "actor_timeout", "2m", "Time to wait on busy actors before timing out")

	case "get":
		usage = "kar get [OPTIONS]"
		description = "Query running application"
		flag.StringVar(&Get, "s", "sidecars", "Information requested")
		flag.StringVar(&OutputStyle, "o", "", "Output style of information calls. 'json' for JSON formatting")

	case "invoke":
		usage = "kar invoke [OPTIONS] ACTOR_TYPE ACTOR_ID METHOD [ARGS]"
		description = "Invoke actor instance"
		Invoke = true

	case "purge":
		usage = "kar purge [OPTIONS]"
		description = "Purge application messages and state"
		Purge = true

	case "drain":
		usage = "kar drain [OPTIONS]"
		description = "Drain application messages"
		Drain = true

	case "-help":
		fallthrough
	case "--help":
		fallthrough
	case "-h":
		fallthrough
	case "--h":
		fallthrough
	case "help":
		writer = os.Stdout
		flag.Usage()
		os.Exit(0)

	default:
		fmt.Fprintln(os.Stderr, "invalid command")
		flag.Usage()
		os.Exit(2)
	}

	// capture command-specific options before adding global options
	b := &strings.Builder{}
	flag.CommandLine.SetOutput(b)
	flag.PrintDefaults()
	options = b.String()
	flag.CommandLine.SetOutput(os.Stderr)

	help := false
	flag.BoolVar(&help, "help", false, "")
	flag.BoolVar(&help, "h", false, "")

	// add global options
	globalOptions(flag.CommandLine)

	flag.CommandLine.Parse(os.Args[2:])

	if help {
		writer = os.Stdout
		flag.Usage()
		os.Exit(0)
	}

	logger.SetVerbosity(verbosity)

	if AppName == "" {
		logger.Fatal("app name is required")
	}

	if ServiceName == "" {
		ServiceName = "kar.none"
	}

	RequestTimeout, err = time.ParseDuration(timeoutTime)
	if err != nil {
		logger.Fatal("error parsing timeout time %s", timeoutTime)
	}

	if CmdName == "run" {
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

		ActorTimeout, err = time.ParseDuration(actorTimeoutTime)
		if err != nil {
			logger.Fatal("error parsing actor timeout time %s", actorTimeoutTime)
		}
	}

	if !KafkaEnableTLS {
		ktmp := os.Getenv("KAFKA_ENABLE_TLS")
		if ktmp == "" {
			ktmp = loadStringFromConfig(configDir, "kafka_enable_tls")
		}
		if ktmp != "" {
			if KafkaEnableTLS, err = strconv.ParseBool(ktmp); err != nil {
				logger.Fatal("error parsing KAFKA_ENABLE_TLS as boolean")
			}
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

	if !RedisEnableTLS {
		rtmp := os.Getenv("REDIS_ENABLE_TLS")
		if rtmp == "" {
			rtmp = loadStringFromConfig(configDir, "redis_enable_tls")
		}
		if rtmp != "" {
			if RedisEnableTLS, err = strconv.ParseBool(rtmp); err != nil {
				logger.Fatal("error parsing REDIS_ENABLE_TLS as boolean")
			}
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

	if Invoke && len(flag.Args()) < 3 {
		logger.Fatal("invoke expects at least three arguments")
	}

	OutputStyle = strings.ToLower(OutputStyle)
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
