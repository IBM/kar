// Package pubsub handles Kafka
package pubsub

import (
	"context"
	"crypto/tls"
	"errors"
	"sync"

	"github.com/Shopify/sarama"
	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

var (
	client   sarama.Client       // shared client
	admin    sarama.ClusterAdmin // shared cluster admin
	producer sarama.SyncProducer // shared idempotent producer

	// routes
	topic    = "kar" + config.Separator + config.AppName
	replicas map[string][]string // map services to sidecars
	hosts    map[string][]string // map actor types to sidecars
	routes   map[string][]int32  // map sidecards to partitions
	tick     = make(chan struct{})
	mu       sync.RWMutex

	// termination
	wg      sync.WaitGroup
	wgMutex sync.Mutex
	wgQuit  bool

	errTooFewPartitions = errors.New("too few partitions")
)

func partitioner(t string) sarama.Partitioner {
	if t == topic {
		return sarama.NewManualPartitioner(t)
	}
	return sarama.NewRandomPartitioner(t)
}

// Dial connects to Kafka
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

	admin, err = sarama.NewClusterAdminFromClient(client)
	if err != nil {
		logger.Debug("failed to instantiate Kafka cluster admin: %v", err)
		return err
	}

	producer, err = sarama.NewSyncProducerFromClient(client)
	if err != nil {
		logger.Debug("failed to instantiate Kafka producer: %v", err)
		return err
	}

	return nil
}

// Close disconnects from Kafka
func Close() {
	wgMutex.Lock()
	wgQuit = true // prevent instantiation of new consumer groups
	wgMutex.Unlock()
	wg.Wait() // wait for all consumer groups to finish
	producer.Close()
	admin.Close()
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
	mu.Lock()
	t := tick
	r := routes[config.ID]
	mu.Unlock()
	return r, t
}

// Join joins the sidecar to the application and returns a channel of incoming messages
func Join(ctx context.Context) (<-chan Message, error) {
	err := admin.CreateTopic(topic, &sarama.TopicDetail{NumPartitions: 1, ReplicationFactor: 3}, false)
	if err != nil {
		err = admin.CreateTopic(topic, &sarama.TopicDetail{NumPartitions: 1, ReplicationFactor: 1}, false)
	}
	if err != nil {
		if e, ok := err.(*sarama.TopicError); !ok || e.Err != sarama.ErrTopicAlreadyExists { // ignore ErrTopicAlreadyExists
			logger.Debug("failed to create Kafka topic: %v", err)
			return nil, err
		}
	}
	return Subscribe(ctx, topic, topic, &Options{master: true, OffsetOldest: true})
}
