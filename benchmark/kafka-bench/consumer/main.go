// Process B
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Shopify/sarama"
	"github.com/prometheus/common/log"
)

var (
	ctx9, cancel9 = context.WithCancel(context.Background()) // preemptive: kill subprocess
	ctx, cancel   = context.WithCancel(ctx9)                 // cooperative: wait for subprocess
)

var kafkaTLSSkipVerify = false
var kafkaEnableTLS = false
var kafkaUsername = ""
var kafkaPassword = ""
var kafkaVersion = ""
var kafkaBrokers = []string{}

// Topics. To be created beforehand using script.
const topic = "simple-topic"
const returnTopic = "return-topic"

// The consumer group on process B.
const group = "simple-consumer-group"

// Repetitions (must match process B's reps):
var warmUpReps = 10
var timedReps = 100

func populateValues() {
	var err error

	if tmp := os.Getenv("KAFKA_BROKERS"); tmp != "" {
		kafkaBrokers = strings.Split(tmp, ",")
	}

	if tmp := os.Getenv("KAFKA_USERNAME"); tmp != "" {
		kafkaUsername = tmp
	}

	if tmp := os.Getenv("KAFKA_PASSWORD"); tmp != "" {
		kafkaPassword = tmp
	}

	if tmp := os.Getenv("KAFKA_USERNAME"); tmp != "" {
		kafkaUsername = tmp
	}
	if kafkaPassword != "" && kafkaUsername == "" {
		kafkaUsername = "token"
	}

	if tmp := os.Getenv("KAFKA_VERSION"); tmp != "" {
		kafkaVersion = tmp
	}

	if tmp := os.Getenv("KAFKA_ENABLE_TLS"); tmp != "" {
		if kafkaEnableTLS, err = strconv.ParseBool(tmp); err != nil {
			fmt.Printf("error parsing KAFKA_TLS_SKIP_VERIFY as boolean")
		}
	}
}

func newConfig() (*sarama.Config, error) {
	populateValues()
	conf := sarama.NewConfig()
	var err error
	conf.Version, err = sarama.ParseKafkaVersion(kafkaVersion)
	if err != nil {
		fmt.Printf("failed to parse Kafka version: %v", err)
		return nil, err
	}
	conf.ClientID = "kar"
	if kafkaPassword != "" {
		conf.Net.SASL.Enable = true
		conf.Net.SASL.User = kafkaUsername
		conf.Net.SASL.Password = kafkaPassword
		conf.Net.SASL.Handshake = true
		conf.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	}
	if kafkaEnableTLS {
		conf.Net.TLS.Enable = true
		// TODO support custom CA certificate
		if kafkaTLSSkipVerify {
			conf.Net.TLS.Config = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
	}
	return conf, nil
}

// handler of consumer group session
type handler struct {
	client   sarama.Client
	conf     *sarama.Config      // kafka config
	topic    string              // subscribed topic
	ready    chan struct{}       // channel closed when ready to accept events
	producer sarama.SyncProducer // The producer used to send back responses
}

func newHandler(conf *sarama.Config, topic string, producer sarama.SyncProducer) *handler {
	return &handler{
		conf:     conf,
		topic:    topic,
		ready:    make(chan struct{}),
		producer: producer,
	}
}

// Setup consumer group session
func (h *handler) Setup(session sarama.ConsumerGroupSession) error {
	log.Info("Inside Setup!")
	close(h.ready)
	// h.ready = make(chan struct{})
	return nil
}

// Cleanup consumer group session
func (h *handler) Cleanup(session sarama.ConsumerGroupSession) error {
	log.Info("Inside Cleanup!")
	return nil
}

// ConsumeClaim processes messages of consumer claim.
func (h *handler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// NOTE:
	// Do not move the code below to a goroutine.
	// The `ConsumeClaim` itself is called within a goroutine, see:
	// https://github.com/Shopify/sarama/blob/master/consumer_group.go#L27-L29
	var count = 0
	var accDuration float64 = 0
	for message := range claim.Messages() {
		// Mark time when message arrived.
		currentTime := time.Now()

		// Send back response with the received time.
		if h.producer != nil {
			msg := &sarama.ProducerMessage{
				Topic:     returnTopic,
				Partition: 0,
				Value:     sarama.StringEncoder(string(message.Value)),
			}
			h.producer.SendMessage(msg)
		}

		// Process request time.
		startTime, err := time.Parse(time.RFC3339Nano, string(message.Value))
		if err != nil {
			log.Info("Time parse error!")
		}
		duration := currentTime.Sub(startTime).Microseconds()
		if count >= warmUpReps {
			accDuration += float64(duration)
			if count == warmUpReps+timedReps-1 {
				log.Infof("Average Kafka request time: %v ms", (accDuration/float64(count))/1000.0)
				count = -1
				accDuration = 0
			}
		}

		// Always increment.
		count++

		// Mark message as processed.
		session.MarkMessage(message, "")
	}

	return nil
}

// Subscribe joins a consumer group and consumes messages on a topic.
func subscribe(ctx context.Context, topic, group string, producer sarama.SyncProducer) (<-chan struct{}, int, error) {
	if ctx.Err() != nil { // fail fast
		return nil, http.StatusServiceUnavailable, ctx.Err()
	}
	conf, err := newConfig()
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	conf.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRange
	conf.Consumer.Offsets.Initial = sarama.OffsetOldest

	handler := newHandler(conf, topic, producer)
	handler.client, err = sarama.NewClient(kafkaBrokers, conf)
	if err != nil {
		log.Error("failed to instantiate Kafka client: %v", err)
		return nil, http.StatusInternalServerError, err
	}

	_, err = handler.client.Partitions(topic)
	if err != nil {
		return nil, http.StatusNotFound, err
	}

	consumer, err := sarama.NewConsumerGroupFromClient(group, handler.client)
	if err != nil {
		log.Error("failed to instantiate Kafka consumer for topic %s, group %s: %v", topic, group, err)
		handler.client.Close()
		return nil, http.StatusInternalServerError, err
	}

	closed := make(chan struct{})

	// consumer loop
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			if err := consumer.Consume(ctx, []string{topic}, handler); err != nil { // abnormal termination
				log.Error("failed Kafka consumer for topic %s, group %s: %T, %#v", topic, group, err, err)
				break
			}
			if ctx.Err() != nil { // normal termination
				break
			}
		}
	}()

	<-handler.ready // Await till the consumer has been set up
	log.Info("Sarama consumer up and running!...")

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-ctx.Done():
		log.Info("terminating: context cancelled")
	case <-sigterm:
		log.Info("terminating: via signal")
	}
	cancel()
	wg.Wait()
	log.Info("terminating: closing handler")
	if err = handler.client.Close(); err != nil {
		log.Info("Error closing handler: %v", err)
	}

	select {
	case <-handler.ready:
	case <-closed:
	}

	return closed, http.StatusOK, nil
}

func createProducer() sarama.SyncProducer {
	config, err := newConfig()
	if err != nil {
		log.Errorf("Error during configuration: %v", err)
	}

	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer(kafkaBrokers, config)
	if err != nil {
		// Should not reach here
		panic(err)
	}

	return producer
}

func main() {
	log.Info("Starting consumer...")
	log.Infof("Kafka brokers is %v", kafkaBrokers)
	// Create the event producer.
	// Set producer to nil if return message is not desired.
	producer := createProducer()

	// Create and subscribe consumer group.
	_, _, err := subscribe(ctx, topic, group, producer)
	if err != nil {
		log.Error("subscribe failed.")
	}
}
