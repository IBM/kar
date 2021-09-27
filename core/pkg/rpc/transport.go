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

package rpc

import (
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
			h.channels[p] = make(chan (<-chan *sarama.ConsumerMessage), 1)
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
	if err := client.RefreshMetadata(appTopic); err != nil {
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
	client.Config().Consumer.Group.Member.UserData, _ = json.Marshal(self)

	if recovery == nil && len(session.Claims()[appTopic]) > 0 { // not in recovery
		mu.Lock() // acquire W mutex to prevent producer from sending
	}

	return nil
}

// Consume messages from claim or run recovery code
func (h *handler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	if recovery != nil {
		// recovery leader, run recovery code
		return h.recover(session, claim)
	}

	// not in recovery (nodes other than the leader are not assigned partitions during recovery)
	for msg := range claim.Messages() {
		if msg.Offset < head {
			continue // skip messages we have already processed
		}
		switch m := decode(msg).(type) {
		case CallRequest:
			if node2partition[m.Caller] != 0 {
				processor(m) // process calls from live nodes only
			}
		case TellRequest, Response:
			processor(m)
		}
		head = msg.Offset + 1
	}
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

	defer close(h.finished)

	requests := map[string]Request{}        // all the non-cancelled requests in disconnected partitions
	requests0 := map[string]struct{}{}      // all the non-cancelled request ids in partition zero
	handled := map[string]struct{}{}        // all the request ids that have a matching response or appear in partitions connected to live nodes
	offsetsForDeletion := map[int32]int64{} // a map from partition to the first offset to preserve in the partition

	// iterate over all claimed partitions and all messages
	for _, p := range session.Claims()[appTopic] {
		messages := <-h.channels[p]
		next := int64(0)
		for next < newest[p] {
			msg := <-messages
			next = msg.Offset + 1
			m := decode(msg)
			switch v := decode(msg).(type) {
			case CallRequest:
				if node2partition[v.Caller] == 0 {
					continue // ignore calls from dead nodes, they are cancelled
				}
				if !recovery[p] { // collect requests targetting dead nodes
					requests[v.requestID()] = v
					if p == 0 {
						requests0[v.requestID()] = struct{}{}
					}
					continue
				}
			case TellRequest:
				if !recovery[p] { // collect requests targetting dead nodes
					requests[v.requestID()] = v
					if p == 0 {
						requests0[v.requestID()] = struct{}{}
					}
					continue
				}
			}
			handled[m.requestID()] = struct{}{} // requests targetting live nodes and responses
		}
		if !recovery[p] && p != 0 { // partition 0 may still contain requests for unavailable services
			offsetsForDeletion[p] = sarama.OffsetNewest
		}
	}

	// refresh topic metadata for producer
	if err := client.RefreshMetadata(appTopic); err != nil {
		return err
	}

	// resend requests targetting dead nodes
	for k, v := range requests {
		// skip requests that are already handled
		if _, ok := handled[k]; !ok {
			// do not send to partition 0 if already in partition 0
			_, ok0 := requests0[k]
			err := resend(session.Context(), v, ok0)
			if err != nil {
				if err != session.Context().Err() {
					logger.Error("resend error during recovery: %v", err)
				}
				return err
			}
		}
	}

	// empty recovered partitions
	admin.DeleteRecords(appTopic, offsetsForDeletion)
	// remember partition 0 offset to avoid an infinite recovery loop
	offset0 = max0
	return nil
}
