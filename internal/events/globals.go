// Package events implements the events API using Kafka
package events

import (
	"crypto/tls"
	"sync"

	"github.com/Shopify/sarama"
	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

var (
	admin    sarama.ClusterAdmin // shared cluster admin
	producer sarama.SyncProducer // shared idempotent producer

	// termination
	mu   sync.Mutex
	wg   sync.WaitGroup
	quit bool
)

// Dial connects to Kafka
func Dial() error {
	// shared cluster admin
	conf, err := newConfig()
	if err != nil {
		return err
	}
	admin, err = sarama.NewClusterAdmin(config.KafkaBrokers, conf)
	if err != nil {
		logger.Debug("failed to instantiate Kafka cluster admin: %v", err)
		return err
	}

	// shared idempotent producer
	conf, err = newConfig()
	if err != nil {
		return err
	}
	conf.Producer.Return.Successes = true
	conf.Producer.RequiredAcks = sarama.WaitForAll
	conf.Producer.Idempotent = true
	conf.Net.MaxOpenRequests = 1
	producer, err = sarama.NewSyncProducer(config.KafkaBrokers, conf)
	if err != nil {
		logger.Debug("failed to instantiate Kafka producer: %v", err)
		return err
	}
	return nil
}

// Close disconnects from Kafka
func Close() {
	mu.Lock()
	quit = true // prevent instantiation of new consumer groups
	mu.Unlock()
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
	conf.ClientID = "kar-events"
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
