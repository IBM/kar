//
// Copyright IBM Corporation 2020,2021
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package pubsub

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"sync"

	"github.com/IBM/kar/core/internal/config"
	"github.com/IBM/kar/core/pkg/logger"
	"github.com/IBM/kar/core/pkg/store"
	"github.com/Shopify/sarama"
)

// store key for topic, partition
func mangle(topic string, partition int32) string {
	return "pubsub" + config.Separator + topic + config.Separator + strconv.Itoa(int(partition))
}

// data exchanged when setting up consumer group session for application topic
type userData struct {
	Address string                       // ip:port of sidecar
	Sidecar string                       // id of this sidecar
	Service string                       // name of this service
	Actors  []string                     // types of actors implemented by this service
	Offsets map[int32]map[int64]struct{} // live local offsets
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

// Mark marks a message as consumed if coming from kafka
func (e *Message) Mark() error {
	if e.handler != nil {
		return e.handler.mark(e.partition, e.offset)
	}
	return nil
}

// handler of consumer group session
type handler struct {
	client     sarama.Client
	conf       *sarama.Config // kafka config
	karContext context.Context
	topic      string                       // subscribed topic
	options    *Options                     // options
	f          func(Message)                // Message handler
	ready      chan struct{}                // channel closed when ready to accept events
	local      map[int32]map[int64]struct{} // local progress: offsets currently worked on in this sidecar
	lock       sync.Mutex                   // mutex to protect local map
	live       map[int32]map[int64]struct{} // offsets in progress at beginning of session (from all sidecars)
	done       map[int32]map[int64]struct{} // offsets completed at beginning of session (from all sidecars)
}

func newHandler(conf *sarama.Config, karContext context.Context, topic string, options *Options, f func(Message)) *handler {
	return &handler{
		conf:       conf,
		karContext: karContext,
		topic:      topic,
		options:    options,
		f:          f,
		ready:      make(chan struct{}),
		local:      map[int32]map[int64]struct{}{},
	}
}

func (h *handler) mark(partition int32, offset int64) error {
	logger.Debug("finishing work on topic %s, partition %d, offset %d", h.topic, partition, offset)
	_, err := store.ZAdd(h.karContext, mangle(h.topic, partition), offset, strconv.FormatInt(offset, 10)) // tell store offset is done first
	if err != nil {
		// TODO retry logic
		logger.Error("failed to mark message on topic %s, partition %d, offset %d: %v", err)
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
			Address: address,
			Sidecar: config.ID,
			Service: config.ServiceName,
			Actors:  config.ActorTypes,
			Offsets: h.local,
		})
	} else {
		h.conf.Consumer.Group.Member.UserData, _ = json.Marshal(h.local) // exchange only local progress
	}
	h.lock.Unlock()
}

// Setup consumer group session
func (h *handler) Setup(session sarama.ConsumerGroupSession) error {
	logger.Info("setup session for topic %s, generation %d, claims %v", h.topic, session.GenerationID(), session.Claims()[h.topic])

	admin, err := sarama.NewClusterAdminFromClient(h.client)
	if err != nil {
		logger.Error("failed to instantiate Kafka cluster admin: %v", err)
		return err
	}

	groups, err := admin.DescribeConsumerGroups([]string{h.topic})
	if err != nil {
		logger.Error("failed to describe consumer group: %v", err)
		return err
	}
	members := groups[0].Members

	var rp map[string][]string // temp replicas
	var hs map[string][]string // temp hosts
	var rt map[string][]int32  // temp routes
	var ad map[string]string   // temp addresses

	if h.options.master {
		for _, member := range members { // ensure enough partitions
			if len(member.MemberAssignment) == 0 { // sidecar without partition
				logger.Info("increasing partition count for topic %s to %d", h.topic, len(members))
				if err := admin.CreatePartitions(h.topic, int32(len(members)), nil, false); err != nil {
					// do not fail if another sidecar added partitions already
					if e, ok := err.(*sarama.TopicPartitionError); !ok || e.Err != sarama.ErrInvalidPartitions {
						logger.Error("failed to add partitions: %v", err)
						return err
					}
				}
				return errTooFewPartitions // abort
			}
		}
		rp = map[string][]string{}
		hs = map[string][]string{}
		rt = map[string][]int32{}
		ad = map[string]string{}
	}

	h.live = map[int32]map[int64]struct{}{} // clear live list
	h.done = map[int32]map[int64]struct{}{} // clear done list

	// populate done map from store
	for _, p := range session.Claims()[h.topic] {
		h.live[p] = map[int64]struct{}{}
		h.done[p] = map[int64]struct{}{}
		r, err := store.ZRange(session.Context(), mangle(h.topic, p), 0, -1) // fetch done offsets from store
		if err != nil {
			logger.Error("failed to retrieve offsets from store: %v", err)
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
			logger.Error("failed to parse member metadata: %v", err)
			return err
		}
		var remote map[int32]map[int64]struct{}
		if h.options.master { // also recompute routes
			var d userData
			if err := json.Unmarshal(m.UserData, &d); err != nil {
				logger.Error("failed to unmarshal user data: %v", err)
				return err
			}
			rp[d.Service] = append(rp[d.Service], d.Sidecar)
			for _, t := range d.Actors {
				hs[t] = append(hs[t], d.Sidecar)
			}
			a, err := member.GetMemberAssignment()
			if err != nil {
				logger.Error("failed to parse member assignment: %v", err)
				return err
			}
			rt[d.Sidecar] = append(rt[d.Sidecar], a.Topics[h.topic]...)
			remote = d.Offsets
			ad[d.Sidecar] = d.Address
		} else {
			if err := json.Unmarshal(m.UserData, &remote); err != nil {
				logger.Error("failed to unmarshal user data: %v", err)
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
			logger.Error("failed to refresh topic: %v", err)
			return err
		}
		mu.Lock()
		replicas = rp
		hosts = hs
		routes = rt
		addresses = ad
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
	store.ZRemRangeByScore(session.Context(), mangle(h.topic, claim.Partition()), 0, claim.InitialOffset()-1) // trim done list
	// ok to ignore error in ZRemRangeByScore as this is just garbage collection
	mark := true
	for m := range claim.Messages() {
		select {
		case <-session.Context().Done():
			return nil // fail fast
		default:
		}
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
		h.f(Message{Value: m.Value, partition: m.Partition, offset: m.Offset, handler: h})
	}
	return nil
}

// Subscribe joins a consumer group and consumes messages on a topic
// f is invoked on each message (serially for each partition)
// f must return quickly if the context is cancelled
func Subscribe(ctx context.Context, topic, group string, options *Options, f func(Message)) (<-chan struct{}, int, error) {
	if ctx.Err() != nil { // fail fast
		return nil, http.StatusServiceUnavailable, ctx.Err()
	}
	conf, err := newConfig()
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	if options.master {
		conf.Consumer.Group.Rebalance.Strategy = &customStrategy{}
	} else {
		conf.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRange
	}
	if options.OffsetOldest {
		conf.Consumer.Offsets.Initial = sarama.OffsetOldest
	}
	handler := newHandler(conf, ctx, topic, options, f)
	handler.marshal()
	handler.client, err = sarama.NewClient(config.KafkaBrokers, conf)
	if err != nil {
		logger.Error("failed to instantiate Kafka client: %v", err)
		return nil, http.StatusInternalServerError, err
	}
	_, err = client.Partitions(topic)
	if err != nil {
		return nil, http.StatusNotFound, err
	}
	consumer, err := sarama.NewConsumerGroupFromClient(group, handler.client)
	if err != nil {
		logger.Error("failed to instantiate Kafka consumer for topic %s, group %s: %v", topic, group, err)
		handler.client.Close()
		return nil, http.StatusInternalServerError, err
	}

	closed := make(chan struct{})

	// consumer loop
	go func() {
		defer close(closed)
		for {
			if err := consumer.Consume(ctx, []string{topic}, handler); err != nil && err != errTooFewPartitions { // abnormal termination
				logger.Error("failed Kafka consumer for topic %s, group %s: %T, %#v", topic, group, err, err)
				// TODO maybe add an error channel
				break
			}
			if ctx.Err() != nil { // normal termination
				break
			}
		}
		consumer.Close()
		handler.client.Close()
	}()

	select {
	case <-handler.ready:
	case <-closed:
	}

	return closed, http.StatusOK, nil
}

type entry struct {
	memberID string
}

type customStrategy struct{}

func (s *customStrategy) Name() string { return "custom" }

func (s *customStrategy) Plan(members map[string]sarama.ConsumerGroupMemberMetadata, topics map[string][]int32) (sarama.BalanceStrategyPlan, error) {
	partitions := topics[topic]

	entries := []entry{}
	for memberID, m := range members {
		var d userData
		json.Unmarshal(m.UserData, &d)
		entries = append(entries, entry{memberID: memberID})
	}

	// TODO: Can this loop be further simplified now that the !e.avoid is gone?
	for i, e := range entries {
		if i != 0 {
			entries[i] = entries[0]
			entries[0] = e
		}
		break
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
