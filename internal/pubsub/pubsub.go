// Package pubsub handles the communication between sidecars.
package pubsub

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"strconv"
	"sync"

	"github.com/Shopify/sarama"
	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/internal/store"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

type userData struct {
	Sidecar string                       // id of this sidecar
	Service string                       // name of this service
	Actors  []string                     // types of actors implemented by this service
	Live    map[int32]map[int64]struct{} // live offsets
}

var (
	conf     = newConfig()
	admin    sarama.ClusterAdmin
	client   sarama.Client // client for producer to control topic refresh
	producer sarama.SyncProducer
	consumer sarama.ConsumerGroup
	topic    = "kar" + config.Separator + config.AppName

	// output channel
	out  = make(chan Message) // TODO multiple channels?
	tick = make(chan struct{})

	// routes
	replicas map[string][]string // map services to sidecars
	hosts    map[string][]string // map actor types to sidecars
	routes   map[string][]int32  // map sidecards to partitions
	mu       = &sync.RWMutex{}   // synchronize changes to routes

	// state of this sidecar
	here = userData{
		Sidecar: config.ID,
		Service: config.ServiceName,
		Actors:  config.ActorTypes,
		Live:    map[int32]map[int64]struct{}{}, // offsets in progress in this sidecar
	}
	lock = &sync.Mutex{} // synchronize updates (Mark is called asynchronously)

	// progress (no lock needed)
	offsets = make([]int64, 1)           // next offset to read from each partition (local uncommitted progress)
	live    map[int32]map[int64]struct{} // live offsets at beginning of session (from all sidecars)
	done    map[int32]map[int64]struct{} // done offsets at beginning of session (from all sidecars)

	errTooFewPartitions = errors.New("too few partitions")
)

// Message is the type of messages
type Message struct {
	Value     map[string]string
	partition int32
	offset    int64
}

func marshal() []byte {
	lock.Lock()
	b, err := json.Marshal(here)
	lock.Unlock()
	if err != nil {
		logger.Fatal("failed to marshal user data: %v", err)
	}
	return b
}

func partitionKey(p int32) string {
	return "pubsub" + config.Separator + "partition" + config.Separator + strconv.Itoa(int(p))
}

func mark(partition int32, offset int64) {
	logger.Debug("finishing work on message at partition %d, offset %d", partition, offset)
	_, err := store.ZAdd(partitionKey(partition), offset, strconv.FormatInt(offset, 10)) // tell store offset is done
	if err != nil {
		logger.Error("failed to add offset to store: %v", err)
	}
	lock.Lock()
	delete(here.Live[partition], offset) // no longer in progress
	lock.Unlock()
}

// Mark marks message as done after processing is complete
func (msg *Message) Mark() {
	mark(msg.partition, msg.offset)
}

// handler of consumer group session
type handler struct{}

// Setup consumer group session
func (consumer *handler) Setup(session sarama.ConsumerGroupSession) error {
	logger.Info("generation %d, sidecar %s, claims %v", session.GenerationID(), here.Sidecar, session.Claims()[topic])
	groups, err := admin.DescribeConsumerGroups([]string{topic})
	if err != nil {
		logger.Debug("failed to describe consumer group: %v", err)
		return err
	}
	members := groups[0].Members
	for _, member := range members { // ensure enough partitions
		if len(member.MemberAssignment) == 0 { // sidecar without partition
			logger.Info("increasing partition count to %d", len(members))
			if err := admin.CreatePartitions(topic, int32(len(members)), nil, false); err != nil {
				// do not fail if another sidecar added partitions already
				if e, ok := err.(*sarama.TopicPartitionError); !ok || e.Err != sarama.ErrInvalidPartitions {
					logger.Debug("failed to add partitions: %v", err)
					return err
				}
			}
			return errTooFewPartitions // abort
		}
	}
	rp := map[string][]string{}           // temp replicas
	hs := map[string][]string{}           // temp hosts
	rt := map[string][]int32{}            // temp routes
	live = map[int32]map[int64]struct{}{} // clear live list
	done = map[int32]map[int64]struct{}{} // clear done list
	max := 0                              // max partition index
	for _, member := range members {
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
		var there userData
		if err := json.Unmarshal(m.UserData, &there); err != nil {
			logger.Fatal("failed to unmarshal user data: %v", err)
		}
		rp[there.Service] = append(rp[there.Service], there.Sidecar)
		for _, t := range there.Actors {
			hs[t] = append(hs[t], there.Sidecar)
		}
		rt[there.Sidecar] = append(rt[there.Sidecar], a.Topics[topic]...)
		for _, p := range session.Claims()[topic] { // for each partition assigned to this sidecar
			if int(p) > max {
				max = int(p)
			}
			if live[p] == nil { // new partition
				live[p] = map[int64]struct{}{}
				done[p] = map[int64]struct{}{}
				lock.Lock()
				here.Live[p] = map[int64]struct{}{}
				lock.Unlock()
			}
			r, err := store.ZRange(partitionKey(p), 0, -1) // fetch done offsets from store
			if err != nil {
				logger.Error("failed to retrieve offsets from store: %v", err)
			}
			for _, o := range r {
				k, _ := strconv.ParseInt(o, 10, 64)
				done[p][k] = struct{}{}
			}
			for k := range there.Live[p] {
				live[p][k] = struct{}{}
			}
		}
	}
	if err := client.RefreshMetadata(topic); err != nil { // refresh producer
		logger.Debug("failed to refresh topic: %v", err)
		return err
	}
	if max >= len(offsets) { // grow array to accommodate more partitions
		offsets = append(offsets, make([]int64, max+1-len(offsets))...)
	}
	mu.Lock()
	replicas = rp
	hosts = hs
	routes = rt
	close(tick)
	tick = make(chan struct{})
	mu.Unlock()
	return nil
}

// Cleanup consumer group session
func (consumer *handler) Cleanup(session sarama.ConsumerGroupSession) error {
	conf.Consumer.Group.Member.UserData = marshal()
	return nil
}

// ConsumeClaim processes messages of consumer claim
func (consumer *handler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	store.ZRemRangeByScore(partitionKey(claim.Partition()), 0, claim.InitialOffset()-1) // trim done list
	advancing := true                                                                   // should advance cursor
	for m := range claim.Messages() {
		logger.Debug("received message at partition %d, offset %d, with value %s", m.Partition, m.Offset, m.Value)
		if _, ok := done[m.Partition][m.Offset]; ok {
			if advancing {
				session.MarkMessage(m, "") // advance cursor
			}
			logger.Debug("skipping committed message at partition %d, offset %d", m.Partition, m.Offset)
			continue
		}
		advancing = false // stop advancing cursor
		if _, ok := live[m.Partition][m.Offset]; ok || offsets[m.Partition] > m.Offset {
			logger.Debug("skipping uncommitted message at partition %d, offset %d", m.Partition, m.Offset)
			continue
		}
		offsets[m.Partition] = m.Offset + 1
		lock.Lock()
		here.Live[m.Partition][m.Offset] = struct{}{}
		lock.Unlock()
		var msg map[string]string
		err := json.Unmarshal(m.Value, &msg)
		if err != nil {
			logger.Error("failed to unmarshal message: %v", err)
			mark(m.Partition, m.Offset) // mark invalid message as completed
			continue
		}
		logger.Debug("starting work on message at partition %d, offset %d", m.Partition, m.Offset)
		select {
		case <-session.Context().Done():
			logger.Debug("cancelling work on message at partition %d, offset %d", m.Partition, m.Offset)
			offsets[m.Partition] = m.Offset // rollback
			lock.Lock()
			delete(here.Live[m.Partition], m.Offset) // rollback
			lock.Unlock()
		case out <- Message{Value: msg, partition: m.Partition, offset: m.Offset}:
		}
	}
	return nil
}

// consume orchestrate the consumer group sessions
func consume(ctx context.Context) {
	for { // for each session
		if err := consumer.Consume(ctx, []string{topic}, &handler{}); err != nil {
			if err != errTooFewPartitions {
				logger.Fatal("consumer error: %v", err)
			}
		}
		if ctx.Err() != nil {
			close(out)
			return // consumer was cancelled
		}
		// next session
	}
}

func newConfig() *sarama.Config {
	conf := sarama.NewConfig()
	if version, err := sarama.ParseKafkaVersion(config.KafkaVersion); err != nil {
		logger.Fatal("invalid Kafka version: %v", err)
	} else {
		conf.Version = version
	}

	conf.ClientID = "kar"

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
	return conf
}

// Dial establishes a connection to Kafka and returns a read channel from incoming messages
func Dial(ctx context.Context) <-chan Message {
	conf.Producer.Return.Successes = true
	conf.Producer.RequiredAcks = sarama.WaitForAll
	conf.Producer.Partitioner = sarama.NewManualPartitioner
	conf.Producer.Idempotent = true
	conf.Net.MaxOpenRequests = 1
	conf.Consumer.Offsets.Initial = sarama.OffsetOldest
	conf.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRange
	conf.Consumer.Group.Member.UserData = marshal()

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
		err = admin.CreateTopic(topic, &sarama.TopicDetail{NumPartitions: 1, ReplicationFactor: 3}, false)
		if err != nil {
			err = admin.CreateTopic(topic, &sarama.TopicDetail{NumPartitions: 1, ReplicationFactor: 1}, false)
		}
		if err != nil {
			// do not fail if another sidecar created the topic already
			if e, ok := err.(*sarama.TopicError); !ok || e.Err != sarama.ErrTopicAlreadyExists {
				logger.Fatal("failed to create Kafka topic: %v", err)
			}
		}
	}

	client, err = sarama.NewClient(config.KafkaBrokers, conf)
	if err != nil {
		logger.Fatal("failed to create Kafka client: %v", err)
	}

	producer, err = sarama.NewSyncProducerFromClient(client)
	if err != nil {
		logger.Fatal("failed to create Kafka producer: %v", err)
	}

	consumer, err = sarama.NewConsumerGroup(config.KafkaBrokers, topic, conf)
	if err != nil {
		logger.Fatal("failed to create Kafka consumer group: %v", err)
	}

	go consume(ctx)

	return out
}

// Partitions returns the set of partitions claimed by this sidecar and a channel for change notifications
func Partitions() ([]int32, <-chan struct{}) {
	mu.Lock()
	t := tick
	r := routes[config.ID]
	mu.Unlock()
	return r, t
}

// Close closes the connection to Kafka
func Close() {
	consumer.Close() // stop accepting incoming messages first
	producer.Close()
	admin.Close()
	client.Close()
}
