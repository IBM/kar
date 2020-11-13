package pubsub

import (
	"net/http"

	"github.com/Shopify/sarama"
	"github.ibm.com/solsa/kar.git/core/pkg/logger"
)

// Publish publishes a message on a topic
func Publish(topic string, message []byte) ( /* httpStatusCode */ int, error) {
	partition, offset, err := producer.SendMessage(&sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(message),
	})
	if err != nil {
		logger.Error("failed to send message on topic %s: %v", topic, err)
		if err == sarama.ErrUnknownTopicOrPartition {
			return http.StatusNotFound, err
		}
		return http.StatusInternalServerError, err
	}
	logger.Debug("sent message on topic %s, partition %d, offset %d", topic, partition, offset)
	return http.StatusOK, nil
}
