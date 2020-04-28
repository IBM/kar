package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"sync"

	"github.com/Shopify/sarama"
	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/internal/store"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

// store key for topic, partition
func mangle(topic string, partition int32) string {
	return "pubsub" + config.Separator + topic + config.Separator + strconv.Itoa(int(partition))
}

// data exchanged when setting up consumer group session for application topic
type userData struct {
	Sidecar                 string   // id of this sidecar
	Service                 string   // name of this service
	Actors                  []string // types of actors implemented by this service
	PartitionZeroIneligible bool
	Offsets                 map[int32]map[int64]struct{} // live local offsets
}

// Options specifies the options for subscribing to a topic
type Options struct {
	OffsetOldest bool // should start from oldest available offset if no cursor exists
	master       bool // internal flag to trigger special handling of application topic
}

// A Message received on a topic
type Message struct {
	Value     []byte   // expose event payload
	partition int32    // hidden
	offset    int64    // hidden
	handler   *handler // hidden
}

// Mark marks a message as consumed
func (e *Message) Mark() error {
	return e.handler.mark(e.partition, e.offset)
}

// handler of consumer group session
type handler struct {
	conf    *sarama.Config               // kafka config
	topic   string                       // subscribed topic
	options *Options                     // options
	out     chan Message                 // output channel
	ready   chan struct{}                // channel closed when ready to accept events
	local   map[int32]map[int64]struct{} // local progress: offsets currently worked on in this sidecar
	lock    sync.Mutex                   // mutex to protect local map
	live    map[int32]map[int64]struct{} // offsets in progress at beginning of session (from all sidecars)
	done    map[int32]map[int64]struct{} // offsets completed at beginning of session (from all sidecars)
}

func newHandler(conf *sarama.Config, topic string, options *Options) *handler {
	return &handler{
		conf:    conf,
		topic:   topic,
		options: options,
		out:     make(chan Message),
		ready:   make(chan struct{}),
		local:   map[int32]map[int64]struct{}{},
	}
}

func (h *handler) mark(partition int32, offset int64) error {
	logger.Debug("finishing work on topic %s, partition %d, offset %d", h.topic, partition, offset)
	_, err := store.ZAdd(mangle(h.topic, partition), offset, strconv.FormatInt(offset, 10)) // tell store offset is done first
	if err != nil {
		logger.Debug("failed to mark message on topic %s, partition %d, offset %d: %v", err)
		return err
	}
	h.lock.Lock()
	delete(h.local[partition], offset) // then remove offset from local map
	h.lock.Unlock()
	return nil
}

// Update member user data
func (h *handler) marshal() {
	h.lock.Lock()
	if h.options.master { // exchange metadata and local progress
		h.conf.Consumer.Group.Member.UserData, _ = json.Marshal(userData{
			Sidecar:                 config.ID,
			Service:                 config.ServiceName,
			Actors:                  config.ActorTypes,
			PartitionZeroIneligible: config.PartitionZeroIneligible,
			Offsets:                 h.local,
		})
	} else {
		h.conf.Consumer.Group.Member.UserData, _ = json.Marshal(h.local) // exchange only local progress
	}
	h.lock.Unlock()
}

// Setup consumer group session
func (h *handler) Setup(session sarama.ConsumerGroupSession) error {
	logger.Info("setup session for topic %s, generation %d, claims %v", h.topic, session.GenerationID(), session.Claims()[h.topic])

	groups, err := admin.DescribeConsumerGroups([]string{h.topic})
	if err != nil {
		logger.Debug("failed to describe consumer group: %v", err)
		return err
	}
	members := groups[0].Members

	var rp map[string][]string // temp replicas
	var hs map[string][]string // temp hosts
	var rt map[string][]int32  // temp routes

	if h.options.master {
		for _, member := range members { // ensure enough partitions
			if len(member.MemberAssignment) == 0 { // sidecar without partition
				logger.Info("increasing partition count for topic %s to %d", h.topic, len(members))
				if err := admin.CreatePartitions(h.topic, int32(len(members)), nil, false); err != nil {
					// do not fail if another sidecar added partitions already
					if e, ok := err.(*sarama.TopicPartitionError); !ok || e.Err != sarama.ErrInvalidPartitions {
						logger.Debug("failed to add partitions: %v", err)
						return err
					}
				}
				return errTooFewPartitions // abort
			}
		}
		rp = map[string][]string{}
		hs = map[string][]string{}
		rt = map[string][]int32{}
	}

	h.live = map[int32]map[int64]struct{}{} // clear live list
	h.done = map[int32]map[int64]struct{}{} // clear done list

	// populate done map from store
	for _, p := range session.Claims()[h.topic] {
		h.live[p] = map[int64]struct{}{}
		h.done[p] = map[int64]struct{}{}
		r, err := store.ZRange(mangle(h.topic, p), 0, -1) // fetch done offsets from store
		if err != nil {
			logger.Debug("failed to retrieve offsets from store: %v", err)
			return err
		}
		for _, o := range r {
			k, _ := strconv.ParseInt(o, 10, 64)
			h.done[p][k] = struct{}{}
		}
	}

	// populate live map from member user data
	for _, member := range members {
		m, err := member.GetMemberMetadata()
		if err != nil {
			logger.Debug("failed to parse member metadata: %v", err)
			return err
		}
		var remote map[int32]map[int64]struct{}
		if h.options.master { // also recompute routes
			var d userData
			if err := json.Unmarshal(m.UserData, &d); err != nil {
				logger.Debug("failed to unmarshal user data: %v", err)
				return err
			}
			rp[d.Service] = append(rp[d.Service], d.Sidecar)
			for _, t := range d.Actors {
				hs[t] = append(hs[t], d.Sidecar)
			}
			a, err := member.GetMemberAssignment()
			if err != nil {
				logger.Debug("failed to parse member assignment: %v", err)
				return err
			}
			rt[d.Sidecar] = append(rt[d.Sidecar], a.Topics[h.topic]...)
			remote = d.Offsets
		} else {
			if err := json.Unmarshal(m.UserData, &remote); err != nil {
				logger.Debug("failed to unmarshal user data: %v", err)
				return err
			}
		}
		for _, p := range session.Claims()[h.topic] {
			for k := range remote[p] {
				h.live[p][k] = struct{}{}
			}
		}
	}

	if h.options.master {
		if err := client.RefreshMetadata(topic); err != nil { // refresh producer
			logger.Debug("failed to refresh topic: %v", err)
			return err
		}
		mu.Lock()
		replicas = rp
		hosts = hs
		routes = rt
		close(tick)
		tick = make(chan struct{})
		mu.Unlock()
	}

	close(h.ready)
	h.ready = make(chan struct{})
	return nil
}

// Cleanup consumer group session
func (h *handler) Cleanup(session sarama.ConsumerGroupSession) error {
	logger.Info("cleanup session for topic %s, generation %d", h.topic, session.GenerationID())
	h.marshal()
	return nil
}

// ConsumeClaim processes messages of consumer claim
func (h *handler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	store.ZRemRangeByScore(mangle(h.topic, claim.Partition()), 0, claim.InitialOffset()-1) // trim done list
	mark := true
	for m := range claim.Messages() {
		logger.Debug("received message on topic %s, partition %d, offset %d", m.Topic, m.Partition, m.Offset)
		if _, ok := h.done[m.Partition][m.Offset]; ok {
			logger.Debug("skipping committed message on topic %s, partition %d, offset %d", m.Topic, m.Partition, m.Offset)
			continue
		}
		if mark { // mark first offset not known to be done
			session.MarkOffset(m.Topic, m.Partition, m.Offset, "")
			mark = false
		}
		if _, ok := h.live[m.Partition][m.Offset]; ok {
			logger.Debug("skipping uncommitted message on topic %s, partition %d, offset %d", m.Topic, m.Partition, m.Offset)
			continue
		}
		h.lock.Lock()
		if h.local[m.Partition] == nil {
			h.local[m.Partition] = map[int64]struct{}{}
		}
		h.local[m.Partition][m.Offset] = struct{}{}
		h.lock.Unlock()
		logger.Debug("starting work on topic %s, at partition %d, offset %d", m.Topic, m.Partition, m.Offset)
		select {
		case <-session.Context().Done():
			logger.Debug("cancelling work on topic %s, partition %d, offset %d", m.Topic, m.Partition, m.Offset)
			h.lock.Lock()
			delete(h.local[m.Partition], m.Offset) // rollback
			h.lock.Unlock()
		case h.out <- Message{Value: m.Value, partition: m.Partition, offset: m.Offset, handler: h}:
		}
	}
	return nil
}

// Subscribe joins a consumer group for a topic
func Subscribe(ctx context.Context, topic, group string, options *Options) (<-chan Message, error) {
	wgMutex.Lock()
	if wgQuit {
		wgMutex.Unlock()
		return nil, fmt.Errorf("failed to instantiate Kafka consumer for topic %s, group %s: shutting down", topic, group)
	}
	wg.Add(1) // increment subscriber count
	wgMutex.Unlock()
	conf, err := newConfig()
	if err != nil {
		wg.Done()
		return nil, err
	}
	if options.master {
		conf.Consumer.Group.Rebalance.Strategy = &customStrategy{}
	} else {
		conf.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRange
	}
	if options.OffsetOldest {
		conf.Consumer.Offsets.Initial = sarama.OffsetOldest
	}
	handler := newHandler(conf, topic, options)
	handler.marshal()
	consumer, err := sarama.NewConsumerGroup(config.KafkaBrokers, group, conf)
	if err != nil {
		wg.Done()
		logger.Debug("failed to instantiate Kafka consumer for topic %s, group %s: %v", topic, group, err)
		return nil, err
	}

	go func() {
		defer wg.Done()
		for {
			if err := consumer.Consume(ctx, []string{topic}, handler); err != nil { // abnormal termination
				if err != errTooFewPartitions || !handler.options.master {
					logger.Error("failed consumer for topic %s, group %s: %v", topic, group, err)
					// TODO maybe emit an error message on output channel before closing
					consumer.Close()
					close(handler.out)
					return
				}
			}
			if ctx.Err() != nil { // normal termination
				consumer.Close()
				close(handler.out)
				return
			}
		}
	}()

	<-handler.ready // wait for first session setup
	return handler.out, nil
}

type entry struct {
	memberID string
	avoid    bool
}

type customStrategy struct{}

func (s *customStrategy) Name() string { return "custom" }

func (s *customStrategy) Plan(members map[string]sarama.ConsumerGroupMemberMetadata, topics map[string][]int32) (sarama.BalanceStrategyPlan, error) {
	partitions := topics[topic]

	entries := []entry{}
	for memberID, m := range members {
		var d userData
		json.Unmarshal(m.UserData, &d)
		entries = append(entries, entry{memberID: memberID, avoid: d.PartitionZeroIneligible})
	}

	for i, e := range entries {
		if !e.avoid {
			if i != 0 {
				entries[i] = entries[0]
				entries[0] = e
			}
			break
		}
	}

	plan := make(sarama.BalanceStrategyPlan, len(members))
	step := float64(len(partitions)) / float64(len(entries))

	for i, e := range entries {
		pos := float64(i)
		min := int(math.Floor(pos*step + 0.5))
		max := int(math.Floor((pos+1)*step + 0.5))
		plan.Add(e.memberID, topic, partitions[min:max]...)
	}
	return plan, nil
}

func (s *customStrategy) AssignmentData(memberID string, topics map[string][]int32, generationID int32) ([]byte, error) {
	return nil, nil
}
