package pubsub

import (
	"crypto/tls"
	"encoding/json"
	"fmt"

	"github.com/Shopify/sarama"
	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

var (
	producer  sarama.SyncProducer
	consumer  sarama.Consumer
	partition sarama.PartitionConsumer
)

// Send sends a message to a service
func Send(service string, message map[string]string) error {
	msg, err := json.Marshal(message)
	if err != nil {
		logger.Error("failed to marshal message %v: %v", message, err)
		return err
	}

	topic := fmt.Sprintf("kar-%s-%s", config.AppName, service)

	partition, offset, err := producer.SendMessage(&sarama.ProducerMessage{
		// TODO Key?
		Topic: topic,
		Value: sarama.ByteEncoder(msg),
	})
	if err != nil {
		logger.Error("failed to send message to topic %s: %v", topic, err)
	}

	logger.Info("sent message on topic %s, at partition %d, offset %d, with value %s", topic, partition, offset, string(msg))
	return nil
}

func receive(in <-chan *sarama.ConsumerMessage, out chan<- map[string]string) {
	for {
		msg, ok := <-in
		if !ok {
			close(out)
			return
		}
		logger.Info("received message on topic %s, at partition %d, offset %d, with value %s", msg.Topic, msg.Partition, msg.Offset, msg.Value)
		var m map[string]string
		err := json.Unmarshal(msg.Value, &m)
		if err != nil {
			logger.Error("ignoring invalid message from topic %s, at partition %d, offset %d: %v", msg.Topic, msg.Partition, msg.Offset, err)
			continue
		}
		out <- m
	}
}

// Dial establishes a connection to Kafka and returns a read channel from incoming messages
func Dial() <-chan map[string]string {
	conf := sarama.NewConfig()

	if version, err := sarama.ParseKafkaVersion(config.KafkaVersion); err != nil {
		logger.Fatal("invalid Kafka version: %v", err)
	} else {
		conf.Version = version
	}

	conf.ClientID = "kar" // TODO
	conf.Producer.Return.Successes = true
	conf.Producer.RequiredAcks = sarama.WaitForAll

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
			InsecureSkipVerify: true, // TODO
		}
	}

	clusterAdmin, err := sarama.NewClusterAdmin(config.KafkaBrokers, conf)
	if err != nil {
		logger.Fatal("failed to create Kafka cluster admin: %v", err)
	}
	defer clusterAdmin.Close()

	topic := fmt.Sprintf("kar-%s-%s", config.AppName, config.ServiceName)

	topics, err := clusterAdmin.ListTopics()
	if err != nil {
		logger.Fatal("failed to list Kafka topics: %v", err)
	}
	if _, ok := topics[topic]; !ok {
		err = clusterAdmin.CreateTopic(topic, &sarama.TopicDetail{NumPartitions: 1, ReplicationFactor: 1}, false)
		if err != nil {
			logger.Fatal("failed to create Kafka topic: %v", err.Error())
		}
	}

	producer, err = sarama.NewSyncProducer(config.KafkaBrokers, conf)
	if err != nil {
		logger.Fatal("failed to create Kafka producer: %v", err)
	}

	consumer, err = sarama.NewConsumer(config.KafkaBrokers, conf)
	if err != nil {
		logger.Fatal("failed to create Kafka consumer: %v", err)
	}

	partition, err = consumer.ConsumePartition(topic, 0, sarama.OffsetNewest) // TODO consumer group
	if err != nil {
		logger.Fatal("failed to create Kafka partition consumer: %v", err)
	}

	out := make(chan map[string]string)
	go receive(partition.Messages(), out)
	return out
}

// Close closes the connection to Kafka
func Close() {
	producer.Close()
	partition.Close()
	consumer.Close()
}
