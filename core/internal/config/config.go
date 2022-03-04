//
// Copyright IBM Corporation 2020,2022
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

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

	"github.com/IBM/kar/core/pkg/logger"
	"github.com/IBM/kar/core/pkg/rpc"
	"github.com/IBM/kar/core/pkg/store"
)

// Separator character for store keys and topic names
const Separator = "_" // must not be a legal DNS name character

var Version = "unofficial"

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
	// VersionCmd is the command "version"
	VersionCmd = "version"
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

	// AppPort is the HTTP port the application process will be listening on
	AppPort int

	// RuntimePort is the HTTP port the runtime will be listening on
	RuntimePort int

	// KubernetesMode is true when this process is running in a sidecar container in a Kubernetes Pod
	KubernetesMode bool

	// KafkaConfig contains the configuration information to connect with Kafka
	KafkaConfig rpc.Config

	// RedisConfig contains the configuration information to connect with Redis
	RedisConfig store.StoreConfig

	// H2C enables h2c to communicate with the app service
	H2C bool

	// Hostname is the name of the host
	Hostname string

	// RequestRetryLimit is how long to retry failing connections in a Redis/http call before giving up
	// A negative time will apply default durations
	RequestRetryLimit time.Duration

	// ActorBusyTimeout is how long to wait on a busy actor instance before timing out and returning an error.
	ActorBusyTimeout time.Duration

	// MissingComponentTimeout is how long to wait on a missing service or actor type before timing out and returning an error.
	MissingComponentTimeout time.Duration

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
	topicConfig                                                   = map[string]*string{"retention.ms": strptr("900000"), "segment.ms": strptr("300000")}

	// enable cache for actor placement
	placementCache bool
)

func strptr(x string) *string { return &x }

// define the flags available on all commands
func globalOptions(f *flag.FlagSet) {
	f.StringVar(&AppName, "app", "", "The name of the application (required)")

	f.StringVar(&kafkaBrokers, "kafka_brokers", "", "The Kafka brokers to connect to, as a comma separated list")
	f.BoolVar(&KafkaConfig.EnableTLS, "kafka_enable_tls", false, "Use TLS to communicate with Kafka")
	f.StringVar(&KafkaConfig.User, "kafka_username", "", "The SASL username if any")
	f.StringVar(&KafkaConfig.Password, "kafka_password", "", "The SASL password if any")
	f.StringVar(&KafkaConfig.Version, "kafka_version", "", "Kafka cluster version")
	f.BoolVar(&KafkaConfig.TLSSkipVerify, "kafka_tls_skip_verify", false, "Skip server name verification for Kafka when connecting over TLS")
	f.Func("kafka_topic_config", "Kafka topic config: k1=v1,k2=v2,...", func(arg string) error {
		for _, x := range strings.Split(arg, ",") {
			kv := strings.Split(x, "=")
			if len(kv) != 2 {
				return fmt.Errorf("kafka_topic_config: ill-formed argument: %v", kv)
			}
			topicConfig[kv[0]] = strptr(kv[1])
		}
		return nil
	})

	f.StringVar(&RedisConfig.Host, "redis_host", "", "The Redis host")
	f.IntVar(&RedisConfig.Port, "redis_port", 0, "The Redis port")
	f.BoolVar(&RedisConfig.EnableTLS, "redis_enable_tls", false, "Use TLS to communicate with Redis")
	f.StringVar(&RedisConfig.Password, "redis_password", "", "The password to use to connect to the Redis server")
	f.StringVar(&RedisConfig.User, "redis_user", "", "The user to use to connect to the Redis server")
	f.BoolVar(&RedisConfig.TLSSkipVerify, "redis_tls_skip_verify", false, "Skip server name verification for Redis when connecting over TLS")
	f.StringVar(&redisCABase64, "redis_ca_cert", "", "The base64-encoded Redis CA certificate if any")

	f.DurationVar(&RequestRetryLimit, "request_retry_limit", -1*time.Second, "Time limit on retrying failing redis/http connections (<0 is infinite)")
	f.DurationVar(&RedisConfig.LongOperation, "redis_slow_op_threshold", 1*time.Second, "Threshold for reporting long-running redis operations")

	f.StringVar(&verbosity, "v", "error", "Logging verbosity")

	f.StringVar(&configDir, "config_dir", "", "Directory containing configuration files")

	f.BoolVar(&placementCache, "placement_cache", false, "Use actor placement cache")
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
  version print version
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
		flag.DurationVar(&ActorBusyTimeout, "actor_busy_timeout", 2*time.Minute, "Time to wait on a busy actor before timing out (0 is infinite)")
		flag.DurationVar(&MissingComponentTimeout, "missing_component_timeout", 2*time.Minute, "Time to wait on request to unknown service or actor type before timing out (0 is infinite)")

	case GetCmd:
		usage = "kar get [OPTIONS]"
		description = "Inspect state of an active application"
		flag.StringVar(&GetSystemComponent, "s", "actors", "Subsystem to query [actors|sidecars]")
		flag.BoolVar(&GetResidentOnly, "mr", false, "Only include memory-resident actor instances")
		flag.StringVar(&GetActorType, "t", "", "Type of the actor instance to get")
		flag.StringVar(&GetActorInstanceID, "i", "", "Instance id of a single actor whose state to get")
		flag.StringVar(&GetOutputStyle, "o", "", "Output style of information calls. 'json' for JSON formatting")

	case InvokeCmd:
		flag.DurationVar(&ActorBusyTimeout, "actor_busy_timeout", 2*time.Minute, "Time to wait on a busy actor before timing out (0 is infinite)")
		flag.DurationVar(&MissingComponentTimeout, "missing_component_timeout", 2*time.Minute, "Time to wait on request to unknown service or actor type before timing out (0 is infinite)")
		usage = "kar invoke [OPTIONS] ACTOR_TYPE ACTOR_ID METHOD [ARGS]"
		description = "Invoke actor instance"

	case RestCmd:
		usage = "kar rest [OPTIONS] REST_METHOD SERVICE_NAME PATH [REQUEST_BODY]"
		description = "Peform a REST operation on a service endpoint"
		flag.StringVar(&RestBodyContentType, "content_type", "application/json", "Content-Type of request body")
		flag.DurationVar(&MissingComponentTimeout, "missing_component_timeout", 2*time.Minute, "Time to wait on request to unknown service or actor type before timing out (0 is infinite)")

	case PurgeCmd:
		usage = "kar purge [OPTIONS]"
		description = "Purge application messages and state"

	case DrainCmd:
		usage = "kar drain [OPTIONS]"
		description = "Drain application messages"

	case VersionCmd:
		fmt.Println(Version)
		os.Exit(0)

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
	logger.SetOutput(os.Stdout)

	if AppName == "" {
		logger.Fatal("app name is required")
	}

	if ServiceName == "" {
		ServiceName = "kar.none"
	}

	rpc.PlacementCache = placementCache

	if actorTypes == "" {
		ActorTypes = make([]string, 0)
	} else {
		ActorTypes = strings.Split(actorTypes, ",")
	}

	if RuntimePort == 0 {
		if ptmp := os.Getenv("KAR_RUNTIME_PORT"); ptmp != "" {
			if RuntimePort, err = strconv.Atoi(ptmp); err != nil {
				logger.Fatal("error parsing KAR_RUNTIME_PORT as an integer")
			}
		}
	}

	if AppPort == 8080 {
		if ptmp := os.Getenv("KAR_APP_PORT"); ptmp != "" {
			if AppPort, err = strconv.Atoi(ptmp); err != nil {
				logger.Fatal("error parsing KAR_APP_PORT as an integer")
			}
		}
	}

	if !KafkaConfig.EnableTLS {
		ktmp := os.Getenv("KAFKA_ENABLE_TLS")
		if ktmp == "" {
			ktmp = loadStringFromConfig(configDir, "kafka_enable_tls")
		}
		if ktmp != "" {
			if KafkaConfig.EnableTLS, err = strconv.ParseBool(ktmp); err != nil {
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

	KafkaConfig.Brokers = strings.Split(kafkaBrokers, ",")

	if KafkaConfig.User == "" {
		if KafkaConfig.User = os.Getenv("KAFKA_USERNAME"); KafkaConfig.User == "" {
			if KafkaConfig.User = loadStringFromConfig(configDir, "kafka_username"); KafkaConfig.User == "" {
				KafkaConfig.User = "token"
			}
		}
	}

	if KafkaConfig.Password == "" {
		if KafkaConfig.Password = os.Getenv("KAFKA_PASSWORD"); KafkaConfig.Password == "" {
			KafkaConfig.Password = loadStringFromConfig(configDir, "kafka_password")
		}
	}

	if KafkaConfig.Version == "" {
		if KafkaConfig.Version = os.Getenv("KAFKA_VERSION"); KafkaConfig.Version == "" {
			if KafkaConfig.Version = loadStringFromConfig(configDir, "kafka_version"); KafkaConfig.Version == "" {
				KafkaConfig.Version = "2.8.1"
			}
		}
	}

	if !KafkaConfig.TLSSkipVerify {
		rtmp := os.Getenv("KAFKA_TLS_SKIP_VERIFY")
		if rtmp == "" {
			rtmp = loadStringFromConfig(configDir, "kafka_tls_skip_verify")
		}
		if rtmp != "" {
			if KafkaConfig.TLSSkipVerify, err = strconv.ParseBool(rtmp); err != nil {
				logger.Fatal("error parsing KAFKA_TLS_SKIP_VERIFY as boolean")
			}
		}
	}

	KafkaConfig.TopicConfig = topicConfig

	if !RedisConfig.EnableTLS {
		rtmp := os.Getenv("REDIS_ENABLE_TLS")
		if rtmp == "" {
			rtmp = loadStringFromConfig(configDir, "redis_enable_tls")
		}
		if rtmp != "" {
			if RedisConfig.EnableTLS, err = strconv.ParseBool(rtmp); err != nil {
				logger.Fatal("error parsing REDIS_ENABLE_TLS as boolean")
			}
		}
	}

	if !RedisConfig.TLSSkipVerify {
		rtmp := os.Getenv("REDIS_TLS_SKIP_VERIFY")
		if rtmp == "" {
			rtmp = loadStringFromConfig(configDir, "redis_tls_skip_verify")
		}
		if rtmp != "" {
			if RedisConfig.TLSSkipVerify, err = strconv.ParseBool(rtmp); err != nil {
				logger.Fatal("error parsing REDIS_TLS_SKIP_VERIFY as boolean")
			}
		}
	}

	if RedisConfig.Host == "" {
		if RedisConfig.Host = os.Getenv("REDIS_HOST"); RedisConfig.Host == "" {
			if RedisConfig.Host = loadStringFromConfig(configDir, "redis_host"); RedisConfig.Host == "" {
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
		RedisConfig.CA, err = x509.ParseCertificate(block.Bytes)
		if err != nil {
			logger.Fatal("error parsing Redis CA certificate: %v", err)
		}
	}

	if RedisConfig.Port == 0 {
		if os.Getenv("REDIS_PORT") != "" {
			if RedisConfig.Port, err = strconv.Atoi(os.Getenv("REDIS_PORT")); err != nil {
				logger.Fatal("error parsing environment variable REDIS_PORT")
			}
		} else {
			if rp := loadStringFromConfig(configDir, "redis_port"); rp != "" {
				if RedisConfig.Port, err = strconv.Atoi(rp); err != nil {
					logger.Fatal("error parsing config value for redis_port: %s", rp)
				}
			} else {
				RedisConfig.Port = 6379
			}
		}
	}

	if RedisConfig.Password == "" {
		if RedisConfig.Password = os.Getenv("REDIS_PASSWORD"); RedisConfig.Password == "" {
			RedisConfig.Password = loadStringFromConfig(configDir, "redis_password")
		}
	}

	if RedisConfig.User == "" {
		if RedisConfig.User = os.Getenv("REDIS_USER"); RedisConfig.User == "" {
			RedisConfig.User = loadStringFromConfig(configDir, "redis_user")
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
