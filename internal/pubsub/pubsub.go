// Package pubsub handles the communication between sidecars.
package pubsub

import (
	"context"
	"crypto/tls"
	"encoding/json"
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
	conf     = sarama.NewConfig()
	admin    sarama.ClusterAdmin
	client   sarama.Client // client for producer to control topic refresh
	producer sarama.SyncProducer
	consumer sarama.ConsumerGroup
	topic    = "kar" + config.Separator + config.AppName

	// output channel
	out = make(chan Message) // TODO multiple channels?

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
		logger.Fatal("failed to marshal userData: %v", err)
	}
	return b
}

func partitionKey(p int32) string {
	return "pubsub" + config.Separator + "partition" + config.Separator + strconv.Itoa(int(p))
}

func mark(partition int32, offset int64) {
	logger.Debug("finishing partition %d offset %d", partition, offset)
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
type handler struct {
	repartitionPending bool
}

type tooFewPartitionsError struct {
	tag struct{}
}

func (e tooFewPartitionsError) Error() string {
	return "too few partitions"
}

// Setup consumer group session
func (consumer *handler) Setup(session sarama.ConsumerGroupSession) error {
	logger.Info("generation %d, member %s, service %s, sidecar %s, claims %v", session.GenerationID(), session.MemberID(), here.Service, here.Sidecar, session.Claims()[topic])
	groups, err := admin.DescribeConsumerGroups([]string{topic})
	if err != nil {
		logger.Debug("failed to describe consumer group: %v", err)
		return err
	}
	members := groups[0].Members
	for _, member := range members { // ensure enough partitions
		if len(member.MemberAssignment) == 0 { // sidecar without partition
			consumer.repartitionPending = true
			logger.Info("increasing partition count to %d", len(members))
			if err := admin.CreatePartitions(topic, int32(len(members)), nil, false); err != nil {
				// do not fail if another sidecar added partitions already
				if e, ok := err.(*sarama.TopicPartitionError); !ok || e.Err != sarama.ErrInvalidPartitions {
					logger.Debug("failed to add partitions: %v", err)
					return err
				}
			}
			return tooFewPartitionsError{} // abort
			return nil
		}
	}
	consumer.repartitionPending = false
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
			logger.Fatal("failed to unmarshal userdata: %v", err)
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
				logger.Error("failed to retrieve mark offsets to store: %v", err)
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
	logger.Info("replicas: %v", rp)
	logger.Info("hosts: %v", hs)
	logger.Info("routes: %v", rt)
	mu.Lock()
	replicas = rp
	hosts = hs
	routes = rt
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
	if consumer.repartitionPending {
		<-session.Context().Done() // wait for repartition
		return nil
	}
	store.ZRemRangeByScore(partitionKey(claim.Partition()), 0, claim.InitialOffset()-1) // trim done list
	prefix := true                                                                      // should advance cursor
	for m := range claim.Messages() {
		logger.Debug("received message on topic %s, at partition %d, offset %d, with value %s", m.Topic, m.Partition, m.Offset, m.Value)
		if _, ok := done[m.Partition][m.Offset]; ok {
			if prefix {
				session.MarkMessage(m, "") // advance cursor
			}
			logger.Debug("skipping done message on topic %s, at partition %d, offset %d, with value %s", m.Topic, m.Partition, m.Offset, m.Value)
			continue
		}
		prefix = false // stop advancing cursor
		if _, ok := live[m.Partition][m.Offset]; ok {
			logger.Debug("skipping live message on topic %s, at partition %d, offset %d, with value %s", m.Topic, m.Partition, m.Offset, m.Value)
			continue
		}
		if offsets[m.Partition] > m.Offset {
			logger.Debug("skipping already seen message on topic %s, at partition %d, offset %d, with value %s", m.Topic, m.Partition, m.Offset, m.Value)
			continue
		}
		offsets[m.Partition] = m.Offset + 1
		lock.Lock()
		here.Live[m.Partition][m.Offset] = struct{}{}
		lock.Unlock()
		var msg map[string]string
		err := json.Unmarshal(m.Value, &msg)
		if err != nil {
			logger.Error("failed to unmarshal message with value %s: %v", m.Value, err)
			mark(m.Partition, m.Offset) // mark invalid message as completed
			continue
		}
		logger.Debug("starting partition %d offset %d", m.Partition, m.Offset)
		select {
		case <-session.Context().Done():
			logger.Debug("cancelling partition %d offset %d", m.Partition, m.Offset)
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
			if _, ok := err.(tooFewPartitionsError); !ok {
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

// Dial establishes a connection to Kafka and returns a read channel from incoming messages
func Dial(ctx context.Context) <-chan Message {
	if version, err := sarama.ParseKafkaVersion(config.KafkaVersion); err != nil {
		logger.Fatal("invalid Kafka version: %v", err)
	} else {
		conf.Version = version
	}

	conf.ClientID = "kar"
	conf.Producer.Return.Successes = true
	conf.Producer.RequiredAcks = sarama.WaitForAll
	conf.Producer.Partitioner = sarama.NewManualPartitioner
	conf.Producer.Idempotent = true
	conf.Net.MaxOpenRequests = 1
	conf.Consumer.Offsets.Initial = sarama.OffsetOldest
	conf.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRange
	conf.Consumer.Group.Member.UserData = marshal()

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

// Close closes the connection to Kafka
func Close() {
	consumer.Close() // stop accepting incoming messages first
	producer.Close()
	admin.Close()
	client.Close()
}
