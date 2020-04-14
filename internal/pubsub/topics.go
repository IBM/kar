package pubsub

import (
	"context"

	"github.com/Shopify/sarama"
	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

var publisher sarama.SyncProducer

func init() {
	c := newConfig()
	c.Producer.Return.Successes = true
	c.Producer.RequiredAcks = sarama.WaitForAll
	c.Net.MaxOpenRequests = 1

	var err error
	publisher, err = sarama.NewSyncProducer(config.KafkaBrokers, c)
	if err != nil {
		logger.Fatal("failed to create Kafka publisher: %v", err)
	}
}

// Publish publishes a message to a topic
func Publish(topic, msg string) (string, error) {
	_, _, err := publisher.SendMessage(&sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(msg),
	})
	if err != nil {
		return "", err
	} else {
		return "OK", nil
	}
}

type RawMessage struct {
	Value   string
	message *sarama.ConsumerMessage
	session sarama.ConsumerGroupSession
}

func (m *RawMessage) Mark() {
	m.session.MarkMessage(m.message, "")
}

type topicHandler struct {
	ch    chan RawMessage
	ready chan struct{}
}

func (handler *topicHandler) Setup(session sarama.ConsumerGroupSession) error {
	close(handler.ready)
	return nil
}

func (handler *topicHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (handler *topicHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		select {
		case handler.ch <- RawMessage{Value: string(msg.Value), message: msg, session: session}:
		case <-session.Context().Done():
		}
	}
	return nil
}

// NewSubscriber subscribes to the specified topic
func NewSubscriber(ctx context.Context, topic, id string, oldest bool) <-chan RawMessage {
	c := newConfig()
	if oldest {
		c.Consumer.Offsets.Initial = sarama.OffsetOldest
	}
	subscriber, err := sarama.NewConsumerGroup(config.KafkaBrokers, id, c)

	if err != nil {
		logger.Fatal("failed to create Kafka subscriber: %v", err)
	}
	handler := topicHandler{ch: make(chan RawMessage), ready: make(chan struct{})}
	go func() {
		for {
			if err := subscriber.Consume(ctx, []string{topic}, &handler); err != nil {
				break
			}
			if ctx.Err() != nil {
				break
			}
			handler.ready = make(chan struct{})
		}
		subscriber.Close()
		close(handler.ch)
	}()
	<-handler.ready
	return handler.ch
}
