// Package config loads the runtime configuration
package config

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
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

const (
	// RunCmd is the command "run"
	RunCmd = "run"
	// GetCmd is the command "get"
	GetCmd = "get"
	// InvokeCmd is the command "invoke"
	InvokeCmd = "invoke"
	// RestCmd is the command "rest"
	RestCmd = "rest"
	// PurgeCmd is the command "purge"
	PurgeCmd = "purge"
	// DrainCmd is the command "drain"
	DrainCmd = "drain"
	// HelpCmd is the command "help"
	HelpCmd = "help"
)

var (
	// CmdName is the top-level command to be executed (purge, run, invoke, etc)
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

	// LongRedisOperation sets a threshold used to report long-running redis operations
	LongRedisOperation time.Duration

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

	// RedisTLSSkipVerify is set to skip server name verification for Redis when connecting over TLS
	RedisTLSSkipVerify bool

	// RedisPassword the the password of the Redis instance (optional)
	RedisPassword string

	// Redis certificate
	RedisCA *x509.Certificate

	// ID is the unique id of this sidecar instance
	ID = uuid.New().String()

	// H2C enables h2c to communicate with the app service
	H2C bool

	// Hostname is the name of the host
	Hostname string

	// RequestTimeout is how long to wait in a Redis/http call before timing out
	// A negative time will apply default durations
	RequestTimeout time.Duration

	// ActorTimeout is how long to wait on a busy actor instance or missing actor type before timing it out and returning an error.
	ActorTimeout time.Duration

	// ServiceTimeout is how long to wait on a busy or missing service before timing it out and returning an error.
	ServiceTimeout time.Duration

	// GetSystemComponent describes what system information to get
	GetSystemComponent string

	// GetResidentOnly include non-memory-resisdent actor instances in get query?
	GetResidentOnly bool

	// GetActorType restrict actor gets to specific type
	GetActorType string

	// GetActorInstanceID is an actor instance whose state will be read
	GetActorInstanceID string

	// GetOutputStyle is whether to print a human readable output, or return a JSON string of data.
	// Currently only applies to calling system/information/
	GetOutputStyle string

	// RestBodyContentType specifies the content type of the request body
	RestBodyContentType string

	// temporary variables to parse command line options
	kafkaBrokers, verbosity, configDir, actorTypes, redisCABase64 string
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
	f.BoolVar(&RedisTLSSkipVerify, "redis_tls_skip_verify", false, "Skip server name verification for Redis when connecting over TLS")
	f.StringVar(&redisCABase64, "redis_ca_cert", "", "The base64-encoded Redis CA certificate if any")

	f.DurationVar(&RequestTimeout, "timeout", -1*time.Second, "Time to wait before timing out calls")
	f.DurationVar(&LongRedisOperation, "redis_slow_op_threshold", 1*time.Second, "Threshold for reporting long-running redis operations")

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
  rest    perform a REST operation on a service endpoint
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

	CmdName = os.Args[1]

	switch CmdName {
	case RunCmd:
		usage = "kar run [OPTIONS] [-- [COMMAND]]"
		description = "Run application component"
		flag.StringVar(&ServiceName, "service", "", "The name of the service provided by this process")
		flag.StringVar(&actorTypes, "actors", "", "The actor types provided by this process, as a comma separated list")
		flag.DurationVar(&ActorCollectorInterval, "actor_collector_interval", 10*time.Second, "Actor collector interval")
		flag.DurationVar(&ActorReminderInterval, "actor_reminder_interval", 100*time.Millisecond, "Actor reminder processing interval")
		flag.DurationVar(&ActorReminderAcceptableDelay, "actor_reminder_acceptable_delay", 3*time.Second, "Threshold at which reminders are logged as being late")
		flag.IntVar(&AppPort, "app_port", 8080, "The port used by KAR to connect to the application")
		flag.IntVar(&RuntimePort, "runtime_port", 0, "The port used by the application to connect to KAR")
		flag.BoolVar(&KubernetesMode, "kubernetes_mode", false, "Running as a sidecar container in a Kubernetes Pod")
		flag.BoolVar(&H2C, "h2c", false, "Use h2c to communicate with service")
		flag.StringVar(&Hostname, "hostname", "localhost", "Hostname")
		flag.DurationVar(&ActorTimeout, "actor_timeout", 2*time.Minute, "Time to wait on busy/unknown actors before timing out")
		flag.DurationVar(&ServiceTimeout, "service_timeout", 2*time.Minute, "Time to wait on busy/unknown service before timing out")

	case GetCmd:
		usage = "kar get [OPTIONS]"
		description = "Inspect state of an active application"
		flag.StringVar(&GetSystemComponent, "s", "actors", "Subsystem to query [actors|sidecars]")
		flag.BoolVar(&GetResidentOnly, "mr", false, "Only include memory-resident actor instances")
		flag.StringVar(&GetActorType, "t", "", "Type of the actor instance to get")
		flag.StringVar(&GetActorInstanceID, "i", "", "Instance id of a single actor whose state to get")
		flag.StringVar(&GetOutputStyle, "o", "", "Output style of information calls. 'json' for JSON formatting")

	case InvokeCmd:
		flag.DurationVar(&ActorTimeout, "actor_timeout", 2*time.Minute, "Time to wait on busy/unknown actors before timing out")
		usage = "kar invoke [OPTIONS] ACTOR_TYPE ACTOR_ID METHOD [ARGS]"
		description = "Invoke actor instance"

	case RestCmd:
		usage = "kar rest [OPTIONS] REST_METHOD SERVICE_NAME PATH [REQUEST_BODY]"
		description = "Peform a REST operation on a service endpoint"
		flag.StringVar(&RestBodyContentType, "content_type", "application/json", "Content-Type of request body")
		flag.DurationVar(&ServiceTimeout, "service_timeout", 2*time.Minute, "Time to wait on busy/unknown service before timing out")

	case PurgeCmd:
		usage = "kar purge [OPTIONS]"
		description = "Purge application messages and state"

	case DrainCmd:
		usage = "kar drain [OPTIONS]"
		description = "Drain application messages"

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

	if actorTypes == "" {
		ActorTypes = make([]string, 0)
	} else {
		ActorTypes = strings.Split(actorTypes, ",")
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

	if !RedisTLSSkipVerify {
		rtmp := os.Getenv("REDIS_TLS_SKIP_VERIFY")
		if rtmp == "" {
			rtmp = loadStringFromConfig(configDir, "redis_tls_skip_verify")
		}
		if rtmp != "" {
			if RedisTLSSkipVerify, err = strconv.ParseBool(rtmp); err != nil {
				logger.Fatal("error parsing REDIS_TLS_SKIP_VERIFY as boolean")
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

	if redisCABase64 == "" {
		if redisCABase64 = os.Getenv("REDIS_CA"); redisCABase64 == "" {
			redisCABase64 = loadStringFromConfig(configDir, "redis_ca")
		}
	}

	if redisCABase64 != "" {
		buf, err := base64.StdEncoding.DecodeString(redisCABase64)
		if err != nil {
			logger.Fatal("error parsing Redis CA certificate: %v", err)
		}

		block, _ := pem.Decode(buf)
		RedisCA, err = x509.ParseCertificate(block.Bytes)
		if err != nil {
			logger.Fatal("error parsing Redis CA certificate: %v", err)
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

	if CmdName == InvokeCmd && len(flag.Args()) < 3 {
		logger.Fatal("invoke expects at least three arguments")
	}

	if CmdName == RestCmd && !(len(flag.Args()) == 3 || len(flag.Args()) == 4) {
		logger.Fatal("rest expects either three or four arguments; got %v", len(flag.Args()))
	}

	GetOutputStyle = strings.ToLower(GetOutputStyle)
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
