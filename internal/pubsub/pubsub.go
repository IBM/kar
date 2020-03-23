package pubsub

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"math/rand"
	"strconv"
	"sync"

	"github.com/Shopify/sarama"
	"github.com/cenkalti/backoff/v4"
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
	mu       = sync.RWMutex{}    // synchronize changes to routes
	leader   = false             // session leader?

	// termination
	ctx context.Context

	// state of this sidecar
	here = userData{
		Sidecar: config.ID,
		Service: config.ServiceName,
		Actors:  config.ActorTypes,
		Live:    map[int32]map[int64]struct{}{},
	}

	// local progress
	local = map[int32]map[int64]struct{}{} // offsets started or finished by this sidecar
	lock  = sync.Mutex{}                   // lock to protect local and here.Live

	// global progress (no lock needed)
	live map[int32]map[int64]struct{}     // live offsets at beginning of session
	done = map[int32]map[int64]struct{}{} // done offsets at beginning of session
)

// Message is the type of messages
type Message struct {
	Value     map[string]string
	Valid     bool
	partition int32
	offset    int64
}

func marshal(ud userData) []byte {
	lock.Lock()
	b, err := json.Marshal(ud)
	lock.Unlock()
	if err != nil {
		logger.Fatal("failed to marshal userData: %v", err)
	}
	return b
}

func mangle(p int32) string {
	return "partition" + config.Separator + strconv.Itoa(int(p))
}

// Mark must be called when a message has been processed
func (msg *Message) Mark() {
	logger.Debug("finishing partition %d offset %d", msg.partition, msg.offset)
	_, err := store.ZAdd(mangle(msg.partition), msg.offset, strconv.FormatInt(msg.offset, 10)) // tell store offset is done
	if err != nil {
		logger.Error("redis error: %v", err)
	}
	lock.Lock()
	delete(here.Live[msg.partition], msg.offset)
	lock.Unlock()
}

// Confirm must be called before processing a message
func (msg *Message) Confirm() bool {
	lock.Lock()
	if local[msg.partition] == nil { // new partition
		local[msg.partition] = map[int64]struct{}{}
		here.Live[msg.partition] = map[int64]struct{}{}
	}
	if _, ok := local[msg.partition][msg.offset]; ok { // already processed or in progress
		lock.Unlock()
		logger.Debug("skipping partition %d offset %d", msg.partition, msg.offset)
		return false
	}
	here.Live[msg.partition][msg.offset] = struct{}{}
	local[msg.partition][msg.offset] = struct{}{}
	lock.Unlock()
	logger.Debug("starting to process partition %d offset %d", msg.partition, msg.offset)
	return true
}

// routeToService maps a service to a partition (keep trying)
func routeToService(service string) (partition int32, err error) {
	err = backoff.Retry(func() error { // keep trying
		mu.RLock()
		sidecars := replicas[service]
		if len(sidecars) == 0 { // no sidecar matching this service
			mu.RUnlock()
			logger.Debug("no sidecar for service %s", service)
			return errors.New("no sidecar for service " + service)
		}
		sidecar := sidecars[rand.Int31n(int32(len(sidecars)))] // select random sidecar from list
		partition, err = routeToSidecar(sidecar)               // map sidecar to partition
		mu.RUnlock()
		return err
	}, backoff.WithContext(backoff.NewExponentialBackOff(), ctx)) // TODO adjust timeout
	return
}

// routeToHost maps an actor type to a sidecar (keep trying)
func routeToHost(t string) (sidecar string, err error) {
	err = backoff.Retry(func() error { // keep trying
		mu.RLock()
		sidecars := hosts[t]
		if len(sidecars) == 0 { // no sidecar matching this service
			mu.RUnlock()
			logger.Debug("no sidecar for actor type %s", t)
			return errors.New("no sidecar for actor type " + t)
		}
		sidecar = sidecars[rand.Int31n(int32(len(sidecars)))] // select random sidecar from list
		mu.RUnlock()
		return err
	}, backoff.WithContext(backoff.NewExponentialBackOff(), ctx)) // TODO adjust timeout
	return
}

// Sidecars returns all the routable sidecars
func Sidecars() []string {
	mu.RLock()
	sidecars := []string{}
	for sidecar := range routes {
		sidecars = append(sidecars, sidecar)
	}
	mu.RUnlock()
	return sidecars
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

// routeToActor maps an actor to a stable sidecar to a partition
// only switching to a new sidecar if the existing sidecar has died
func routeToActor(t, id string) (partition int32, sidecar string, err error) {
	key := "actor" + config.Separator + "sidecar" + config.Separator + t + config.Separator + id
	for { // keep trying
		sidecar, err = store.Get(key) // retrieve already assigned sidecar if any
		if err != nil && err != store.ErrNil {
			return // redis error
		}
		if sidecar != "" { // sidecar is already assigned
			_, err = routeToHost(t) // make sure routes have been initialized
			if err != nil {
				return // no matching host, abort
			}
			partition, err = routeToSidecar(sidecar) // find partition for sidecar
			if err == nil {
				return // found sidecar and partition
			}
		}
		expected := sidecar           // prepare to assign new sidecar
		sidecar, err = routeToHost(t) // wait for matching service
		if err != nil {
			return // no matching host, abort
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
	var partition int32
	var err error
	switch message["protocol"] {
	case "service": // route to service
		partition, err = routeToService(message["to"])
		if err != nil {
			logger.Debug("failed to route to service %s: %v", message["to"], err)
			return err
		}
	case "actor": // route to actor
		var sidecar string
		partition, sidecar, err = routeToActor(message["to"], message["id"])
		if err != nil {
			logger.Debug("failed to route to actor type %s id $s %v: %v", message["to"], message["id"], err)
			return err
		}
		message["sidecar"] = sidecar
	case "sidecar": // route to sidecar
		partition, err = routeToSidecar(message["to"])
		if err != nil {
			logger.Debug("failed to route to sidecar %s: %v", message["to"], err)
			return err
		}
	}
	msg, err := json.Marshal(message)
	if err != nil {
		logger.Debug("failed to marshal message with value %v: %v", message, err)
		return err
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
			if leader {
				logger.Info("increasing partition count to %d", len(members))
				if err := admin.CreatePartitions(topic, int32(len(members)), nil, false); err != nil {
					// do not fail if another sidecar added partitions already
					if e, ok := err.(*sarama.TopicPartitionError); !ok || e.Err != sarama.ErrInvalidPartitions {
						logger.Debug("failed to add partitions: %v", err)
						return err
					}
				}
				return tooFewPartitionsError{} // abort
			}
			return nil
		}
	}
	consumer.repartitionPending = false
	rp := map[string][]string{}           // temp replicas
	hs := map[string][]string{}           // temp hosts
	rt := map[string][]int32{}            // temp routes
	live = map[int32]map[int64]struct{}{} // clear live list
	done = map[int32]map[int64]struct{}{} // clear done list
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
			if live[p] == nil { // new partition
				live[p] = map[int64]struct{}{}
				done[p] = map[int64]struct{}{}
			}
			r, err := store.ZRange(mangle(p), 0, -1) // fetch done offsets from store
			if err != nil {
				logger.Error("redis error: %v", err)
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
	leader = false
	conf.Consumer.Group.Member.UserData = marshal(here)
	return nil
}

// ConsumeClaim processes messages of consumer claim
func (consumer *handler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	if consumer.repartitionPending {
		<-session.Context().Done() // wait for repartition
		return nil
	}
	store.ZRemRangeByScore(mangle(claim.Partition()), 0, claim.InitialOffset()-1) // trim done list
	prefix := true                                                                // should advance cursor
	for msg := range claim.Messages() {
		logger.Debug("received message on topic %s, at partition %d, offset %d, with value %s", msg.Topic, msg.Partition, msg.Offset, msg.Value)
		if _, ok := live[msg.Partition][msg.Offset]; ok {
			prefix = false // stop advancing cursor
			logger.Debug("skipping live message on topic %s, at partition %d, offset %d, with value %s", msg.Topic, msg.Partition, msg.Offset, msg.Value)
			continue
		}
		if _, ok := done[msg.Partition][msg.Offset]; ok {
			if prefix {
				session.MarkMessage(msg, "") // advance cursor
			}
			logger.Debug("skipping done message on topic %s, at partition %d, offset %d, with value %s", msg.Topic, msg.Partition, msg.Offset, msg.Value)
			continue
		}
		prefix = false
		var message map[string]string
		err := json.Unmarshal(msg.Value, &message)
		if err != nil {
			logger.Error("failed to unmarshal message with value %s: %v", msg.Value, err)
			continue
		}
		valid := false
		switch message["protocol"] {
		case "service":
			valid = message["to"] == here.Service
		case "actor":
			valid = message["sidecar"] == here.Sidecar
		case "sidecar":
			valid = message["to"] == here.Sidecar
		}
		select {
		case <-session.Context().Done():
		case out <- Message{Value: message, Valid: valid, partition: msg.Partition, offset: msg.Offset}:
		}
	}
	return nil
}

// consume orchestrate the consumer group sessions
func consume() {
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

// setContext sets the global context
func setContext(c context.Context) {
	ctx = c
}

// Wrap balance strategy to detect session leader
type balanceStrategy struct {
	strategy sarama.BalanceStrategy
}

func (s *balanceStrategy) Name() string { return s.strategy.Name() }

func (s *balanceStrategy) Plan(members map[string]sarama.ConsumerGroupMemberMetadata, topics map[string][]int32) (sarama.BalanceStrategyPlan, error) {
	leader = true
	return s.strategy.Plan(members, topics)
}

func (s *balanceStrategy) AssignmentData(memberID string, topics map[string][]int32, generationID int32) ([]byte, error) {
	return s.strategy.AssignmentData(memberID, topics, generationID)
}

// Dial establishes a connection to Kafka and returns a read channel from incoming messages
func Dial(ctx context.Context) <-chan Message {
	setContext(ctx)

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
	conf.Consumer.Group.Rebalance.Strategy = &balanceStrategy{sarama.BalanceStrategyRange}
	conf.Consumer.Group.Member.UserData = marshal(here)

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

	go consume()

	return out
}

// Close closes the connection to Kafka
func Close() {
	consumer.Close() // stop accepting incoming messages first
	producer.Close()
	admin.Close()
	client.Close()
}
