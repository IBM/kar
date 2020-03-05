package pubsub

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"

	"github.com/Shopify/sarama"
	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

var (
	producer sarama.SyncProducer
	consumer sarama.ConsumerGroup
)

// mangle app and service names into topic name
func mangle(app, service string) string {
	return fmt.Sprintf("kar-%s-%s", app, service)
}

// Send sends a message to a service
func Send(service string, message map[string]string) error {
	msg, err := json.Marshal(message)
	if err != nil {
		logger.Error("failed to marshal message %v: %v", message, err)
		return err
	}

	topic := mangle(config.AppName, service)

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

type handler struct {
	out chan map[string]string
}

func (consumer *handler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (consumer *handler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (consumer *handler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		logger.Info("received message on topic %s, at partition %d, offset %d, with value %s", msg.Topic, msg.Partition, msg.Offset, msg.Value)
		session.MarkMessage(msg, "")
		var m map[string]string
		err := json.Unmarshal(msg.Value, &m)
		if err != nil {
			logger.Error("ignoring invalid message from topic %s, at partition %d, offset %d: %v", msg.Topic, msg.Partition, msg.Offset, err)
			continue
		}
		consumer.out <- m
	}
	close(consumer.out)
	return nil
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
	conf.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategySticky

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

	topic := mangle(config.AppName, config.ServiceName)

	topics, err := clusterAdmin.ListTopics()
	if err != nil {
		logger.Fatal("failed to list Kafka topics: %v", err)
	}
	if _, ok := topics[topic]; !ok {
		err = clusterAdmin.CreateTopic(topic, &sarama.TopicDetail{NumPartitions: 1, ReplicationFactor: 1}, false) // TODO
		if err != nil {
			logger.Fatal("failed to create Kafka topic: %v", err.Error())
		}
	}

	producer, err = sarama.NewSyncProducer(config.KafkaBrokers, conf)
	if err != nil {
		logger.Fatal("failed to create Kafka producer: %v", err)
	}

	consumer, err = sarama.NewConsumerGroup(config.KafkaBrokers, topic, conf)
	if err != nil {
		logger.Fatal("failed to create Kafka consumer group: %v", err)
	}

	out := make(chan map[string]string)
	go consumer.Consume(context.Background(), []string{topic}, &handler{out: out})
	return out
}

// Close closes the connection to Kafka
func Close() {
	producer.Close()
	consumer.Close()
}
