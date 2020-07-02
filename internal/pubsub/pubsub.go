// Package pubsub handles Kafka
package pubsub

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"strconv"
	"sync"

	"github.com/Shopify/sarama"
	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

var (
	client   sarama.Client       // shared client
	producer sarama.SyncProducer // shared idempotent producer

	// routes
	topic     = "kar" + config.Separator + config.AppName
	replicas  map[string][]string // map services to sidecars
	hosts     map[string][]string // map actor types to sidecars
	routes    map[string][]int32  // map sidecards to partitions
	address   string              // host:port of sidecar http server (for peer-to-peer connections)
	addresses map[string]string   // map sidecards to addresses
	tick      = make(chan struct{})
	joined    = tick
	mu        = &sync.RWMutex{}

	manualPartitioner = sarama.NewManualPartitioner(topic)

	errTooFewPartitions = errors.New("too few partitions")

	// ErrUnknownSidecar error
	ErrUnknownSidecar = errors.New("unknown sidecar")
)

func partitioner(t string) sarama.Partitioner {
	if t == topic {
		return manualPartitioner
	}
	return sarama.NewRandomPartitioner(t)
}

// Dial connects Kafka producer
func Dial() error {
	conf, err := newConfig()
	if err != nil {
		return err
	}

	conf.Producer.Return.Successes = true
	conf.Producer.RequiredAcks = sarama.WaitForAll
	conf.Producer.Idempotent = true
	conf.Producer.Partitioner = partitioner
	conf.Net.MaxOpenRequests = 1

	client, err = sarama.NewClient(config.KafkaBrokers, conf)
	if err != nil {
		logger.Debug("failed to instantiate Kafka client: %v", err)
		return err
	}

	producer, err = sarama.NewSyncProducerFromClient(client)
	if err != nil {
		logger.Debug("failed to instantiate Kafka producer: %v", err)
		return err
	}

	return nil
}

// Close closes Kafka producer
func Close() {
	producer.Close()
	client.Close()
}

func newConfig() (*sarama.Config, error) {
	conf := sarama.NewConfig()
	var err error
	conf.Version, err = sarama.ParseKafkaVersion(config.KafkaVersion)
	if err != nil {
		logger.Debug("failed to parse Kafka version: %v", err)
		return nil, err
	}
	conf.ClientID = "kar"
	if config.KafkaPassword != "" {
		conf.Net.SASL.Enable = true
		conf.Net.SASL.User = config.KafkaUsername
		conf.Net.SASL.Password = config.KafkaPassword
		conf.Net.SASL.Handshake = true
		conf.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	}
	if config.KafkaEnableTLS {
		conf.Net.TLS.Enable = true
		conf.Net.TLS.Config = &tls.Config{
			InsecureSkipVerify: true, // TODO certificates
		}
	}
	return conf, nil
}

// Partitions returns the set of partitions claimed by this sidecar and a channel for change notifications
func Partitions() ([]int32, <-chan struct{}) {
	mu.RLock()
	t := tick
	r := routes[config.ID]
	mu.RUnlock()
	return r, t
}

// Join joins the sidecar to the application and returns a channel of incoming messages
func Join(ctx context.Context, f func(Message), port int) (<-chan struct{}, error) {
	address = net.JoinHostPort(config.Hostname, strconv.Itoa(port))
	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		logger.Debug("failed to instantiate Kafka cluster admin: %v", err)
		return nil, err
	}
	err = admin.CreateTopic(topic, &sarama.TopicDetail{NumPartitions: 1, ReplicationFactor: 3}, false)
	if err != nil {
		err = admin.CreateTopic(topic, &sarama.TopicDetail{NumPartitions: 1, ReplicationFactor: 1}, false)
	}
	if err != nil {
		if e, ok := err.(*sarama.TopicError); !ok || e.Err != sarama.ErrTopicAlreadyExists { // ignore ErrTopicAlreadyExists
			logger.Debug("failed to create Kafka topic: %v", err)
			return nil, err
		}
	}
	return Subscribe(ctx, topic, topic, &Options{master: true, OffsetOldest: true}, f)
}

// Purge deletes the application topic
func Purge() error {
	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		return err
	}
	err = admin.DeleteTopic(topic)
	if err != sarama.ErrUnknownTopicOrPartition {
		return err
	}
	return nil
}
