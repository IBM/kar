// Process A
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/IBM/kar/core/pkg/logger"
	"github.com/Shopify/sarama"
)

var (
	ctx9, cancel9 = context.WithCancel(context.Background()) // preemptive: kill subprocess
	ctx, cancel   = context.WithCancel(ctx9)                 // cooperative: wait for subprocess
)

var kafkaEnableTLS = false
var kafkaTLSSkipVerify = false
var kafkaUsername = ""
var kafkaPassword = ""
var kafkaVersion = ""
var kafkaBrokers = []string{"localhost:31093"}

// Topics. To be created beforehand using script.
const topic = "simple-topic"
const returnTopic = "return-topic"

// The consumer group on process A.
const returnGroup = "return-consumer-group"

// Repetitions (must match process B's reps):
var warmUpReps = 100
var timedReps = 10000

var endToEndTimings = []float64{}

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
			logger.Fatal("error parsing KAFKA_TLS_SKIP_VERIFY as boolean")
		}
	}
	logger.Info("Kafka brokers is %v", kafkaBrokers)
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
	client sarama.Client
	conf   *sarama.Config // kafka config
	topic  string         // subscribed topic
	ready  chan struct{}  // channel closed when ready to accept events
}

func newHandler(conf *sarama.Config, topic string) *handler {
	return &handler{
		conf:  conf,
		topic: topic,
		ready: make(chan struct{}),
	}
}

// Setup consumer group session
func (h *handler) Setup(session sarama.ConsumerGroupSession) error {
	logger.Info("Inside Setup!")
	close(h.ready)
	// h.ready = make(chan struct{})
	return nil
}

// Cleanup consumer group session
func (h *handler) Cleanup(session sarama.ConsumerGroupSession) error {
	logger.Info("Inside Cleanup!")
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
		// Current time.
		currentTime := time.Now()

		// Process duration.
		startTime, err := time.Parse(time.RFC3339Nano, string(message.Value))
		if err != nil {
			logger.Info("Time parse error!")
		}
		duration := currentTime.Sub(startTime).Microseconds()

		// Post-process durations.
		count++
		if count > warmUpReps {
			// Save duration data.
			endToEndTimings = append(endToEndTimings, float64(duration)/1000.0)

			// At the end print the average.
			accDuration += float64(duration) / 1000.0
			if count == warmUpReps+timedReps {
				logger.Info("Average Kafka end-to-end time: %v ms", (accDuration / float64(timedReps)))
				count = 0
				accDuration = 0
			}
		}

		// Mark message as processed.
		session.MarkMessage(message, "")
	}

	return nil
}

// Subscribe joins a consumer group and consumes messages on a topic.
func subscribe(ctx context.Context, topic, group string) (*sync.WaitGroup, *handler, <-chan struct{}, int, error) {
	if ctx.Err() != nil { // fail fast
		return nil, nil, nil, http.StatusServiceUnavailable, ctx.Err()
	}
	conf, err := newConfig()
	if err != nil {
		return nil, nil, nil, http.StatusInternalServerError, err
	}

	conf.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRange
	conf.Consumer.Offsets.Initial = sarama.OffsetOldest

	handler := newHandler(conf, topic)
	handler.client, err = sarama.NewClient(kafkaBrokers, conf)
	if err != nil {
		logger.Error("failed to instantiate Kafka client: %v", err)
		return nil, nil, nil, http.StatusInternalServerError, err
	}

	_, err = handler.client.Partitions(topic)
	if err != nil {
		return nil, nil, nil, http.StatusNotFound, err
	}

	consumer, err := sarama.NewConsumerGroupFromClient(group, handler.client)
	if err != nil {
		logger.Error("failed to instantiate Kafka consumer for topic %s, group %s: %v", topic, group, err)
		handler.client.Close()
		return nil, nil, nil, http.StatusInternalServerError, err
	}

	closed := make(chan struct{})

	// consumer loop
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			if err := consumer.Consume(ctx, []string{topic}, handler); err != nil { // abnormal termination
				logger.Error("failed Kafka consumer for topic %s, group %s: %T, %#v", topic, group, err, err)
				break
			}
			if ctx.Err() != nil { // normal termination
				break
			}
		}
	}()

	<-handler.ready // Await till the consumer has been set up
	logger.Info("Sarama return consumer up and running!...")

	return wg, handler, closed, http.StatusOK, nil
}

func createProducer() sarama.SyncProducer {
	config, err := newConfig()
	if err != nil {
		logger.Error("Error during configuration: %v", err)
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

func average(data []float64) float64 {
	// Compute sum.
	var sum float64 = 0
	for _, elem := range data {
		sum += elem
	}

	// Compute average:
	return sum / float64(len(data))
}

func orderStats(data []float64) (float64, float64, float64) {
	sort.Float64s(data)
	if len(data) > 99 {
		return data[len(data)/2], data[len(data)*9/10], data[len(data)*99/100]
	} else if len(data) >= 10 {
		return data[len(data)/2], data[len(data)*9/10], data[len(data)-1]
	} else {
		return data[len(data)/2], data[len(data)-1], data[len(data)-1]
	}
}

func printTimingReport(data []float64) {
	// Assumption: input is a list of ms durations.
	avg := average(data)

	// Compute standard deviation
	var deviations = []float64{}
	for _, time := range data {
		deviation := time - avg
		deviations = append(deviations, deviation*deviation)
	}
	standardDeviation := math.Sqrt(average(deviations))

	// Print report:
	median, nine, ninetynine := orderStats(data)
	logger.Info(`Kafka: end-to-end: samples = %v; mean = %v; median = %v; 90th = %v; 99th= %v; stddev = %v`, len(data), avg, median, nine, ninetynine, standardDeviation)
}

func main() {
	logger.SetVerbosity("Info")

	// Create and subscribe consumer group.
	wg, handler, closed, _, err := subscribe(ctx, returnTopic, returnGroup)
	if err != nil {
		logger.Error("subscribe failed.")
	}

	// Create the event producer.
	producer := createProducer()

	defer func() {
		if err := producer.Close(); err != nil {
			// Should not reach here
			panic(err)
		}
	}()

	var partition int32 = 0
	var offset int64 = 0
	for i := 0; i < warmUpReps+timedReps; i++ {
		startTime := time.Now()
		msg := &sarama.ProducerMessage{
			Topic:     topic,
			Partition: 0,
			Value:     sarama.StringEncoder(string(startTime.Format(time.RFC3339Nano))),
		}
		partition, offset, err = producer.SendMessage(msg)
		if err != nil {
			panic(err)
		}

		time.Sleep(50 * time.Millisecond)
	}

	fmt.Printf("Message is stored in topic(%s)/partition(%d)/offset(%d)\n", topic, partition, offset)

	printTimingReport(endToEndTimings)

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-ctx.Done():
		logger.Info("terminating: context cancelled")
	case <-sigterm:
		logger.Info("terminating: via signal")
	}
	cancel()

	wg.Wait()
	logger.Info("terminating: closing handler")
	if err = handler.client.Close(); err != nil {
		logger.Info("Error closing handler: %v", err)
	}

	select {
	case <-handler.ready:
	case <-closed:
	}
}
