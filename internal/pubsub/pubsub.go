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
	"github.ibm.com/solsa/kar.git/internal/actors"
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
	leader   = false             // session leader?

	// termination
	ctx context.Context

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

func mangle(p int32) string {
	return "pubsub" + config.Separator + "partition" + config.Separator + strconv.Itoa(int(p))
}

func mark(partition int32, offset int64) {
	logger.Debug("finishing partition %d offset %d", partition, offset)
	_, err := store.ZAdd(mangle(partition), offset, strconv.FormatInt(offset, 10)) // tell store offset is done
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
		if len(sidecars) == 0 { // no sidecar matching this actor type
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
func routeToActor(actor actors.Actor) (partition int32, sidecar string, err error) {
	for { // keep trying
		sidecar, err = actors.Get(actor) // retrieve already assigned sidecar if any
		if err != nil && err != store.ErrNil {
			return // redis error
		}
		if sidecar != "" { // sidecar is already assigned
			_, err = routeToHost(actor.Type) // make sure routes have been initialized
			if err != nil {
				return // no matching host, abort
			}
			partition, err = routeToSidecar(sidecar) // find partition for sidecar
			if err == nil {
				return // found sidecar and partition
			}
		}
		expected := sidecar                    // prepare to assign new sidecar
		sidecar, err = routeToHost(actor.Type) // wait for matching service
		if err != nil {
			return // no matching host, abort
		}
		_, err = actors.Update(actor, expected, sidecar) // try saving sidecar
		if err != nil {
			return // redis error
		}
		// loop around
	}
}

// Send sends message to receiver
func Send(msg map[string]string) error {
	var partition int32
	var err error
	switch msg["protocol"] {
	case "service": // route to service
		partition, err = routeToService(msg["service"])
		if err != nil {
			logger.Debug("failed to route to service %s: %v", msg["service"], err)
			return err
		}
	case "actor": // route to actor
		var sidecar string
		partition, sidecar, err = routeToActor(actors.Actor{Type: msg["type"], ID: msg["id"]})
		if err != nil {
			logger.Debug("failed to route to actor type %s id $s %v: %v", msg["type"], msg["id"], err)
			return err
		}
		msg["sidecar"] = sidecar // mutate msg
	case "sidecar": // route to sidecar
		partition, err = routeToSidecar(msg["sidecar"])
		if err != nil {
			logger.Debug("failed to route to sidecar %s: %v", msg["sidecar"], err)
			return err
		}
	}
	m, err := json.Marshal(msg)
	if err != nil {
		logger.Debug("failed to marshal message with value %v: %v", msg, err)
		return err
	}
	_, offset, err := producer.SendMessage(&sarama.ProducerMessage{
		Topic:     topic,
		Partition: partition,
		Value:     sarama.ByteEncoder(m),
	})
	if err != nil {
		logger.Debug("failed to send message to partition %d: %v", partition, err)
		return err
	}
	logger.Debug("sent message on topic %s, at partition %d, offset %d, with value %s", topic, partition, offset, string(m))
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
			r, err := store.ZRange(mangle(p), 0, -1) // fetch done offsets from store
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
	leader = false
	conf.Consumer.Group.Member.UserData = marshal()
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

// wrap balance strategy to detect session leader
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
