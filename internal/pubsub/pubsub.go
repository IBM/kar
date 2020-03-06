package pubsub

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"math/rand"
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
	topic    = "kar-" + config.AppName

	// output channel
	out = make(chan map[string]string) // TODO multiple channels?

	// routes
	routes map[string][]int32 // map services to partitions
	mu     = sync.RWMutex{}   // synchronize changes to routes

	ctx, cancel = context.WithCancel(context.Background())
	wg          = sync.WaitGroup{}
)

// Send sends a message to a service
func Send(service string, message map[string]string) error {
	msg, err := json.Marshal(message)
	if err != nil {
		logger.Error("failed to marshal message %v: %v", message, err)
		return err
	}

	// wait for route
	var p []int32
	err = backoff.Retry(func() error {
		var ok bool
		mu.RLock()
		defer mu.RUnlock()
		if p, ok = routes[service]; ok {
			return nil
		}
		return errors.New("")
	}, backoff.NewExponentialBackOff()) // TODO fix timeout
	if err != nil {
		logger.Error("failed to route message to service %s: %v", service, err)
	}

	partition, offset, err := producer.SendMessage(&sarama.ProducerMessage{
		Topic:     topic,
		Partition: p[rand.Int31n(int32(len(p)))],
		Value:     sarama.ByteEncoder(msg),
	})
	if err != nil {
		logger.Error("failed to send message to topic %s: %v", topic, err)
	} else {
		logger.Info("sent message on topic %s, at partition %d, offset %d, with value %s", topic, partition, offset, string(msg))
	}
	return nil
}

type handler struct{}

func (consumer *handler) Setup(session sarama.ConsumerGroupSession) error {
	r := map[string][]int32{}
	groups, _ := admin.DescribeConsumerGroups([]string{topic})
	members := groups[0].Members
	for _, member := range members {
		a, _ := member.GetMemberAssignment()
		m, _ := member.GetMemberMetadata()
		service := string(m.UserData)
		r[service] = append(r[service], a.Topics[topic]...)
	}
	me, _ := members[session.MemberID()].GetMemberAssignment()
	logger.Info("new partitions: %v and routes: %v", me.Topics[topic], r)
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
		logger.Info("received message on topic %s, at partition %d, offset %d, with value %s", msg.Topic, msg.Partition, msg.Offset, msg.Value)
		session.MarkMessage(msg, "")
		var m map[string]string
		err := json.Unmarshal(msg.Value, &m)
		if err != nil {
			logger.Error("ignoring invalid message from topic %s, at partition %d, offset %d: %v", msg.Topic, msg.Partition, msg.Offset, err)
			continue
		}
		out <- m
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
			return
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
	conf.Consumer.Group.Member.UserData = []byte(config.ServiceName)

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
	producer.Close()
	cancel()
	wg.Wait()
	consumer.Close()
	admin.Close()
	close(out)
}
