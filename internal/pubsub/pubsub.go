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
	"github.ibm.com/solsa/kar.git/internal/store"
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
	me       = config.ID + config.Separator + config.ServiceName
	replicas map[string][]string // map services to sidecars
	routes   map[string][]int32  // map sidecards to partitions
	mu       = sync.RWMutex{}    // synchronize changes to routes

	// termination
	ctx context.Context
)

// routeToService maps a service to a partition (keep trying)
func routeToService(service string) (partition int32, sidecar string, err error) {
	err = backoff.Retry(func() error { // keep trying
		mu.RLock()
		sidecars := replicas[service]
		if len(sidecars) == 0 { // no sidecar matching this service
			mu.RUnlock()
			logger.Debug("no sidecar for service %s", service)
			return errors.New("no sidecar for service " + service)
		}
		sidecar = sidecars[rand.Int31n(int32(len(sidecars)))] // select random sidecar from list
		partition, err = routeToSidecar(sidecar)              // map sidecar to partition
		mu.RUnlock()
		return err
	}, backoff.WithContext(backoff.NewExponentialBackOff(), ctx)) // TODO adjust timeout
	return
}

// routeToSidecar maps a sidecar to a partition (no retries)
func routeToSidecar(sidecar string) (int32, error) {
	mu.RLock()
	partitions := routes[sidecar]
	mu.RUnlock()
	if len(partitions) == 0 { // no partition matching this sidecar
		logger.Debug("no partition for sidecar %s", sidecar)
		return -1, errors.New("no partition for sidecar " + sidecar)
	}
	return partitions[rand.Int31n(int32(len(partitions)))], nil // select random partition from list
}

// routeToSession maps a session of a service to a stable sidecar to a partition
// only switching to a new sidecar if the existing sidecar has died
func routeToSession(service string, session string) (partition int32, err error) {
	key := "session" + config.Separator + service + config.Separator + session
	var sidecar string
	for { // keep trying
		sidecar, err = store.Get(key) // retrieve already assigned sidecar if any
		if err != nil && err != store.ErrNil {
			return // redis error
		}
		if sidecar != "" { // no assigned sidecar yet
			_, _, err = routeToService(service) // make sure routes have been initialized
			if err != nil {
				return // no matching service, abort
			}
			partition, err = routeToSidecar(sidecar) // find partition for sidecar
			if err == nil {
				return // found sidecar and partition
			}
		}
		expected := sidecar                       // prepare to update assigned sidecar
		_, sidecar, err = routeToService(service) // wait for matching service
		if err != nil {
			return // no matching service, abort
		}
		_, err = store.CompareAndSet(key, expected, sidecar) // try saving sidecar
		if err != nil {
			return // redis error
		}
		// loop around
	}
}

// Send sends message to receiver
func Send(message map[string]string) error {
	msg, err := json.Marshal(message)
	if err != nil {
		logger.Debug("failed to marshal message with value %v: %v", message, err)
		return err
	}
	var partition int32
	switch message["protocol"] {
	case "service": // route to service
		if message["session"] != "" {
			partition, err = routeToSession(message["to"], message["session"])
		} else {
			partition, _, err = routeToService(message["to"])
		}
		if err != nil {
			logger.Debug("failed to route to service %s%s%s: %v", message["to"], config.Separator, message["session"], err)
			return err
		}
	case "sidecar": // route to sidecar
		partition, err = routeToSidecar(message["to"])
		if err != nil {
			logger.Debug("failed to route to sidecar %s: %v", message["to"], err)
			return err
		}
	}
	_, offset, err := producer.SendMessage(&sarama.ProducerMessage{
		Topic:     topic,
		Partition: partition,
		Value:     sarama.ByteEncoder(msg),
	})
	if err != nil {
		logger.Debug("failed to send message to partition %d: %v", partition, err)
		return err
	}
	logger.Debug("sent message on topic %s, at partition %d, offset %d, with value %s", topic, partition, offset, string(msg))
	return nil
}

// handler of consumer group session
type handler struct{}

// Setup consumer group session
func (consumer *handler) Setup(session sarama.ConsumerGroupSession) error {
	rp := map[string][]string{} // temp replicas
	rt := map[string][]int32{}  // temp routes
	groups, err := admin.DescribeConsumerGroups([]string{topic})
	if err != nil {
		logger.Debug("failed to describe consumer group: %v", err)
		return err
	}
	members := groups[0].Members
	for memberID, member := range members {
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
		data := strings.Split(string(m.UserData), config.Separator)
		sidecar := data[0]
		service := data[1]
		rp[service] = append(rp[service], sidecar)
		rt[sidecar] = append(rt[sidecar], a.Topics[topic]...)
		if memberID == session.MemberID() {
			logger.Info("partitions: %v", a.Topics[topic])
		}
	}
	logger.Info("replicas: %v", rp)
	logger.Info("routes: %v", rt)
	mu.Lock()
	replicas = rp
	routes = rt
	mu.Unlock()
	return nil
}

// Cleanup consumer group session
func (consumer *handler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim processes messages of consumer claim
func (consumer *handler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		session.MarkMessage(msg, "received") // TODO
		logger.Debug("received message on topic %s, at partition %d, offset %d, with value %s", msg.Topic, msg.Partition, msg.Offset, msg.Value)
		var message map[string]string
		err := json.Unmarshal(msg.Value, &message)
		if err != nil {
			logger.Error("failed to unmarshal message with value %s: %v", msg.Value, err)
			continue
		}
		if strings.Contains(me, message["to"]) { // message reached destination
			select {
			case <-session.Context().Done():
			case out <- message:
			}
		} else { // message reached wrong sidecar
			switch message["protocol"] {
			case "service": // route to service
				logger.Info("forwarding message to service %s%s%s", message["to"], config.Separator, message["session"])
			case "sidecar": // route to sidecar
				logger.Info("forwarding message to sidecar %s", message["to"])
			}
			if err := Send(message); err != nil {
				switch message["protocol"] {
				case "service": // route to service
					logger.Error("failed to forward message to service %s%s%s: %v", message["to"], config.Separator, message["session"], err)
				case "sidecar": // route to sidecar
					logger.Debug("failed to forward message to sidecar %s: %v", message["to"], err) // not an error
				}
			}
		}
	}
	return nil
}

// consume orchestrate the consumer group sessions
func consume() {
	for { // for each session
		if err := consumer.Consume(ctx, []string{topic}, &handler{}); err != nil {
			logger.Fatal("consumer error: %v", err)
		}
		if ctx.Err() != nil {
			close(out)
			return // consumer was cancelled
		}
		// next session
	}
}

// setContext sets the global context
func setContext(c context.Context) {
	ctx = c
}

// Dial establishes a connection to Kafka and returns a read channel from incoming messages
func Dial(ctx context.Context) <-chan map[string]string {
	setContext(ctx)
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

	go consume()

	return out
}

// Close closes the connection to Kafka
func Close() {
	consumer.Close() // stop accepting incoming messages first
	producer.Close()
	admin.Close()
}
