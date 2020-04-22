package events

import (
	"github.com/Shopify/sarama"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

// Publish publishes a message on a topic
func Publish(topic string, message []byte) error {
	partition, offset, err := producer.SendMessage(&sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(message),
	})
	if err != nil {
		logger.Debug("failed to send message on topic %s: %v", topic, err)
		return err
	}
	logger.Debug("send message on topic %s, partition %d, offset %d", topic, partition, offset)
	return nil
}
