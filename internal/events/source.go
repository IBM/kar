package events

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/Shopify/sarama"
	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/internal/store"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

func mangle(topic string, partition int32) string {
	return "events" + config.Separator + topic + config.Separator + strconv.Itoa(int(partition))
}

// An Event received for a topic
type Event struct {
	Value     []byte
	partition int32
	offset    int64
	handler   *handler
}

// Mark marks a message as consumed
func (e *Event) Mark() error {
	return e.handler.mark(e.partition, e.offset)
}

// handler of consumer group session
type handler struct {
	conf  *sarama.Config               // kafka config
	topic string                       // subscribed topic
	out   chan Event                   // output channel
	ready chan struct{}                // channel closed when ready to accept events
	local map[int32]map[int64]struct{} // local progress: offsets currently worked on in this sidecar
	lock  sync.Mutex                   // mutex to protect local map
	live  map[int32]map[int64]struct{} // offsets in progress at beginning of session (from all sidecars)
	done  map[int32]map[int64]struct{} // offsets completed at beginning of session (from all sidecars)
}

func newHandler(conf *sarama.Config, topic string) *handler {
	return &handler{
		conf:  conf,
		topic: topic,
		out:   make(chan Event),
		ready: make(chan struct{}),
		local: map[int32]map[int64]struct{}{},
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
	h.conf.Consumer.Group.Member.UserData, _ = json.Marshal(h.local)
	h.lock.Unlock()
}

// Setup consumer group session
func (h *handler) Setup(session sarama.ConsumerGroupSession) error {
	logger.Info("setup session for topic %s, generation %d, claims %v", h.topic, session.GenerationID(), session.Claims()[h.topic])
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
	groups, err := admin.DescribeConsumerGroups([]string{h.topic})
	if err != nil {
		logger.Debug("failed to describe consumer group: %v", err)
		return err
	}
	for _, member := range groups[0].Members {
		m, err := member.GetMemberMetadata()
		if err != nil {
			logger.Debug("failed to parse member metadata: %v", err)
			return err
		}
		var remote map[int32]map[int64]struct{}
		if err := json.Unmarshal(m.UserData, &remote); err != nil {
			logger.Debug("failed to unmarshal user data: %v", err)
			return err
		}
		for _, p := range session.Claims()[h.topic] {
			for k := range remote[p] {
				h.live[p][k] = struct{}{}
			}
		}
	}
	close(h.ready)
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
		case h.out <- Event{Value: m.Value, partition: m.Partition, offset: m.Offset, handler: h}:
		}
	}
	return nil
}

// Subscribe joins a consumer group for a topic
func Subscribe(ctx context.Context, topic, group string, oldest bool) (<-chan Event, error) {
	mu.Lock()
	if quit {
		mu.Unlock()
		return nil, fmt.Errorf("failed to instantiate Kafka consumer for topic %s, group %s: shutting down", topic, group)
	}
	wg.Add(1) // increment subscriber count
	mu.Unlock()
	conf, err := newConfig()
	if err != nil {
		wg.Done()
		return nil, err
	}
	if oldest {
		conf.Consumer.Offsets.Initial = sarama.OffsetOldest
	}
	handler := newHandler(conf, topic)
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
				logger.Error("failed consumer for topic %s, group %s: %v", topic, group, err)
				// TODO maybe emit an error message on output channel before closing
				consumer.Close()
				close(handler.out)
				return
			}
			if ctx.Err() != nil { // normal termination
				consumer.Close()
				close(handler.out)
				return
			}
			handler.ready = make(chan struct{}) // replace channel so it can be closed again
		}
	}()

	<-handler.ready // wait for first session setup
	return handler.out, nil
}
