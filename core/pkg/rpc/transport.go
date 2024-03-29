//
// Copyright IBM Corporation 2020,2023
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

package rpc

import (
	"context"
	"encoding/json"

	"github.com/IBM/kar/core/pkg/logger"
	"github.com/Shopify/sarama"
)

// Consumer group handler
type handler struct {
	channels map[int32]chan (<-chan *sarama.ConsumerMessage) // map to collect all the claims in recovery
	finished chan struct{}                                   // channel to synchronize termination of all the claim consumers in recovery
}

// Setup consumer group session, assumes W mutex is held on entry
func (h *handler) Setup(session sarama.ConsumerGroupSession) error {
	if len(session.Claims()[appTopic]) == 0 { // in recovery, but not leader, nothing to do
		logger.Info("waiting for recovery, generation %d, claims %v", session.GenerationID(), session.Claims()[appTopic])
		return nil // keep mutex
	}

	if recovery != nil { // recovery leader
		logger.Info("leading recovery, generation %d, claims %v", session.GenerationID(), session.Claims()[appTopic])

		// initialize map
		h.channels = map[int32]chan (<-chan *sarama.ConsumerMessage){}
		for _, p := range session.Claims()[appTopic] {
			h.channels[p] = make(chan (<-chan *sarama.ConsumerMessage), 1) // do not block producer
		}

		// initialize channel
		h.finished = make(chan struct{})

		return nil // keep mutex
	}
	// not in recovery, each node has been assigned one partition
	logger.Info("processing messages, generation %d, claims %v", session.GenerationID(), session.Claims()[appTopic])
	self.Partition = session.Claims()[appTopic][0]

	// update service2nodes and node2partitions
	err := updateRoutes()
	if err != nil {
		return err // drop from consumer group if an error occurred
	}

	// refresh topic metadata for producer
	if err := producerClient.RefreshMetadata(appTopic); err != nil {
		return err
	}

	// signal and release mutex on successful setup to resume producer activity
	close(tick)
	tick = make(chan struct{})
	mu.Unlock()
	return nil
}

// Cleanup consumer group session, assumes W mutex is held on entry iff in recovery
func (*handler) Cleanup(session sarama.ConsumerGroupSession) error {
	logger.Info("completed generation %d", session.GenerationID())

	// marshal latest info (to share our assigned partition with others if decided)
	consumerClient.Config().Consumer.Group.Member.UserData, _ = json.Marshal(self)

	if recovery == nil && len(session.Claims()[appTopic]) > 0 { // not in recovery
		mu.Lock() // acquire W mutex to prevent producer from sending
	}
	logger.Info("finish cleanup %v", session.GenerationID())
	return nil
}

// Consume messages from claim or run recovery code
func (h *handler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	if recovery != nil {
		// recovery leader, run recovery code
		return h.recover(session, claim)
	}

	logger.Info("begin claim %v %v", session.GenerationID(), claim.Partition())

	// not in recovery (nodes other than the leader are not assigned partitions during recovery)
	for msg := range claim.Messages() {
		if msg.Offset < head {
			continue // skip messages we have already processed
		}
		switch m := decode(msg).(type) {
		case CallRequest:
			processor(m)
		case TellRequest, Response:
			processor(m)
		}
		head = msg.Offset + 1
	}
	logger.Info("finish claim %v %v", session.GenerationID(), claim.Partition())
	return nil
}

// recovery code
func (h *handler) recover(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// the consumer for partition 0 does most of the work
	// other consumers delegate to the consumer for partition 0 and wait until it finishes

	p := claim.Partition()

	h.channels[p] <- claim.Messages()

	if p != 0 {
		<-h.finished
		return nil
	}

	logger.Info("enter recover %v %v", session.GenerationID(), claim.Partition())

	defer close(h.finished)

	orphans := []Message{}                  // all the messages in dead partitions in order
	orphans0 := []Message{}                 // all the requests in partition 0 in order
	calls := map[string][]string{}          // map caller id to callee ids for blocking calls
	responses := map[string]struct{}{}      // all the response ids
	requests := map[string]struct{}{}       // all the request ids in live partitions or partition 0
	responses0 := map[string]struct{}{}     // all the response ids in partition 0
	requests0 := map[string]int{}           // map request id to max sequence in partition 0
	handled := map[string]int{}             // all the request ids that have a matching response or appear in partitions connected to live nodes
	offsetsForDeletion := map[int32]int64{} // a map from partition to the first offset to preserve in the partition
	min := map[string]int{}
	max := map[string]int{}
	latest := map[string]Message{}
	chains := map[string]map[int]struct{}{}

	// iterate over all claimed partitions and all messages
	for _, p := range session.Claims()[appTopic] {
		var messages <-chan *sarama.ConsumerMessage
		select {
		case messages = <-h.channels[p]:
		case <-session.Context().Done():
			return session.Context().Err()
		}
		next := int64(0)
		for next < newest[p] {
			msg := <-messages
			if msg == nil { // session has been interrupted
				return context.Canceled
			}
			next = msg.Offset + 1
			m := decode(msg)
			switch v := decode(msg).(type) {
			case Request:
				if max[v.requestID()] <= v.sequence() {
					max[v.requestID()] = v.sequence()
					latest[v.requestID()] = v
				}
				if s, ok := v.target().(Session); ok && s.DeferredLockID != "" {
					if chains[v.requestID()] == nil {
						chains[v.requestID()] = map[int]struct{}{}
					}
					chains[v.requestID()][v.sequence()] = struct{}{}
				}
				if c, ok := v.(CallRequest); ok {
					calls[c.ParentID] = append(calls[c.ParentID], c.RequestID)
				}
				if !recovery[p] { // collect requests targetting dead partitions and partition 0
					if p == 0 {
						orphans0 = append(orphans0, v)
						requests[v.requestID()] = struct{}{}
						if v.sequence() >= requests0[m.requestID()] {
							requests0[m.requestID()] = v.sequence()
						}
					} else {
						orphans = append(orphans, v)
					}
					continue
				}
				if v.sequence() >= handled[m.requestID()] {
					handled[m.requestID()] = v.sequence() // requests targetting live partitions
				}
				requests[v.requestID()] = struct{}{}

			default:
				responses[v.requestID()] = struct{}{}
				handled[m.requestID()] = 1 << 30 // responses
				if !recovery[p] && p != 0 {      // collect responses targetting dead partitions
					orphans = append(orphans, v)
				}
				if p == 0 {
					responses0[v.requestID()] = struct{}{}
				}
			}
		}
		if !recovery[p] && p != 0 { // partition 0 may still contain requests for unavailable services
			offsetsForDeletion[p] = sarama.OffsetNewest
		}
	}

	// refresh topic metadata for producer
	if err := producerClient.RefreshMetadata(appTopic); err != nil {
		return err
	}

	for k, v := range max {
		min[k] = v
	}

	for k, v := range chains {
		for i := max[k]; i >= 0; i-- {
			if _, ok := v[i]; !ok {
				min[k] = i
				break
			}
		}
	}

	seen := map[string]struct{}{}

	orphans = append(orphans, orphans0...)

	logger.Info("recover done reading %v %v", session.GenerationID(), claim.Partition())

	// resend messages targetting dead nodes
	for _, msg := range orphans {
		k := msg.requestID()
		switch v := msg.(type) {
		case Request:
			if s, ok := handled[k]; (!ok || s < max[k]) && v.sequence() == min[k] {
				if _, ok := seen[k]; ok {
					continue
				}
				seen[k] = struct{}{}
				childID := ""
				for _, r := range calls[k] { // iterate of nested blocking calls
					if _, ok := responses[r]; !ok { // nested call has not completed
						childID = r
					}
				}
				switch w := latest[k].(type) {
				case CallRequest:
					w.ChildID = childID
					w.Sequence = min[k]
					if t, ok := w.Target.(Session); ok {
						t.DeferredLockID = ""
						w.Target = t
					}
					v = w
				case TellRequest:
					w.ChildID = childID
					w.Sequence = min[k]
					if t, ok := w.Target.(Session); ok {
						t.DeferredLockID = ""
						w.Target = t
					}
					v = w
				}
				// do not send to partition 0 if already in partition 0
				s0, ok0 := requests0[k]
				err := resend(session.Context(), v, ok0 && s0 >= max[k])
				if err != nil {
					if err != session.Context().Err() {
						logger.Error("resend error during recovery: %v", err)
					}
					return err
				}
			}
		default:
			if _, ok := requests[k]; ok {
				if _, ok := responses0[k]; !ok {
					err := respond(session.Context(), Done{RequestID: k, Deadline: v.deadline()})
					if err != nil {
						if err != session.Context().Err() {
							logger.Error("resend error during recovery: %v", err)
						}
						return err
					}
				}
			}
		}
	}

	// empty recovered partitions
	admin.DeleteRecords(appTopic, offsetsForDeletion)
	// remember partition 0 offset to avoid an infinite recovery loop
	offset0 = max0

	logger.Info("exit recover %v %v", session.GenerationID(), claim.Partition())

	return nil
}
