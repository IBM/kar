package pubsub

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"math/rand"
	"strings"
	"sync"

	"github.com/Shopify/sarama"
	"github.com/cenkalti/backoff/v4"
	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

var (
	admin    sarama.ClusterAdmin
	producer sarama.SyncProducer
	consumer sarama.ConsumerGroup
	topic    = "kar" + config.Separator + config.AppName

	// output channel
	out = make(chan map[string]string) // TODO multiple channels?

	// routes
	me     = config.ServiceName + "," + config.ID
	routes map[string][]int32 // map services to partitions
	mu     = sync.RWMutex{}   // synchronize changes to routes

	ctx, cancel = context.WithCancel(context.Background())
	wg          = sync.WaitGroup{}
)

// Send sends message to receiver
func Send(message map[string]string) error {
	msg, err := json.Marshal(message)
	if err != nil {
		logger.Debug("failed to marshal message with value %v: %v", message, err)
		return err
	}

	var p []int32
	switch message["protocol"] {
	case "service": // wait for route to specified service name
		err = backoff.Retry(func() error {
			var ok bool
			mu.RLock()
			defer mu.RUnlock()
			if p, ok = routes[message["to"]]; ok {
				return nil
			}
			logger.Debug("no route to service %s", message["to"]) // not an error if transient
			return errors.New("no route to service " + message["to"])
		}, backoff.NewExponentialBackOff()) // TODO fix timeout
	case "sidecar": // route to specified sidecard uuid if available or fail
		var ok bool
		mu.RLock()
		defer mu.RUnlock()
		if p, ok = routes[message["to"]]; !ok {
			err = errors.New("no route to sidecar " + message["to"]) // no retry
		}
	}
	if err != nil {
		logger.Debug("failed to send message to %s %s: %v", message["protocol"], message["to"], err)
		return err
	}

	partition, offset, err := producer.SendMessage(&sarama.ProducerMessage{
		Topic:     topic,
		Partition: p[rand.Int31n(int32(len(p)))],
		Value:     sarama.ByteEncoder(msg),
	})
	if err != nil {
		logger.Debug("failed to send message to %s %s: %v", message["protocol"], message["to"], err)
		return err
	}

	logger.Debug("sent message on topic %s, at partition %d, offset %d, with value %s", topic, partition, offset, string(msg))

	return nil
}

type handler struct{}

func (consumer *handler) Setup(session sarama.ConsumerGroupSession) error {
	r := map[string][]int32{}
	groups, err := admin.DescribeConsumerGroups([]string{topic})
	if err != nil {
		logger.Debug("failed to describe consumer group: %v", err)
		return err
	}
	members := groups[0].Members
	for id, member := range members {
		a, err := member.GetMemberAssignment()
		if err != nil {
			logger.Debug("failed to parse member assignment: %v", err)
			return err
		}
		m, err := member.GetMemberMetadata()
		if err != nil {
			logger.Debug("failed to parse member metadata: %v", err)
			return err
		}
		services := strings.Split(string(m.UserData), ",")
		for _, service := range services {
			r[service] = append(r[service], a.Topics[topic]...)
		}
		if id == session.MemberID() {
			logger.Info("partitions: %v", a.Topics[topic])
		}
	}
	logger.Info("routes: %v", r)
	mu.Lock()
	defer mu.Unlock()
	routes = r
	return nil
}

func (consumer *handler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (consumer *handler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		session.MarkMessage(msg, "received") // TODO
		logger.Debug("received message on topic %s, at partition %d, offset %d, with value %s", msg.Topic, msg.Partition, msg.Offset, msg.Value)
		var m map[string]string
		err := json.Unmarshal(msg.Value, &m)
		if err != nil {
			logger.Error("failed to unmarshal message with value %s: %v", msg.Value, err)
			continue
		}
		if strings.Contains(me, m["to"]) {
			out <- m
		} else {
			logger.Info("forwarding message to %s %s", m["protocol"], m["to"])
			if err := Send(m); err != nil {
				if m["protocol"] == "service" {
					logger.Error("failed to forward message to service %s: %v", m["to"], err) // error
				} else {
					logger.Debug("failed to forward message to sidecar %s: %v", m["to"], err) // not an error
				}
			}
		}
	}
	return nil
}

func consume() {
	defer wg.Done()
	for {
		if err := consumer.Consume(ctx, []string{topic}, &handler{}); err != nil {
			logger.Fatal("consumer error: %v", err)
		}
		if ctx.Err() != nil {
			return // cancelled
		}
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

	conf.ClientID = "kar"
	conf.Producer.Return.Successes = true
	conf.Producer.RequiredAcks = sarama.WaitForAll
	conf.Producer.Partitioner = sarama.NewManualPartitioner
	conf.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRange
	conf.Consumer.Group.Member.UserData = []byte(me)

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

	var err error

	admin, err = sarama.NewClusterAdmin(config.KafkaBrokers, conf)
	if err != nil {
		logger.Fatal("failed to create Kafka cluster admin: %v", err)
	}

	topics, err := admin.ListTopics()
	if err != nil {
		logger.Fatal("failed to list Kafka topics: %v", err)
	}
	if _, ok := topics[topic]; !ok {
		err = admin.CreateTopic(topic, &sarama.TopicDetail{NumPartitions: 10, ReplicationFactor: 3}, false) // TODO fix NumPartitions
		if err != nil {
			err = admin.CreateTopic(topic, &sarama.TopicDetail{NumPartitions: 10, ReplicationFactor: 1}, false)
		}
		if err != nil {
			logger.Fatal("failed to create Kafka topic: %v", err)
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

	wg.Add(1)
	go consume()

	return out
}

// Close closes the connection to Kafka
func Close() {
	cancel()
	wg.Wait()
	consumer.Close()
	producer.Close()
	admin.Close()
	close(out)
}
