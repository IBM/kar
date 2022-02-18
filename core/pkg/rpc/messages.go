//
// Copyright IBM Corporation 2020,2022
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
	"strconv"
	"time"

	"github.com/Shopify/sarama"
)

type Message interface {
	requestID() string
	value() []byte
	deadline() time.Time
}

type Request interface {
	Message
	target() Target
	method() string
	sequence() int
}

type CallRequest struct {
	RequestID string    // request id
	Value     []byte    // payload
	Deadline  time.Time // deadline to start executing request
	Target    Target    // target
	Method    string    // method
	Caller    string    // source node
	Sequence  int       // sequence number
}

func (m CallRequest) requestID() string   { return m.RequestID }
func (m CallRequest) value() []byte       { return m.Value }
func (m CallRequest) deadline() time.Time { return m.Deadline }
func (m CallRequest) target() Target      { return m.Target }
func (m CallRequest) method() string      { return m.Method }
func (m CallRequest) sequence() int       { return m.Sequence }

type TellRequest struct {
	RequestID string    // request id
	Value     []byte    // payload
	Deadline  time.Time // deadline to start executing request
	Target    Target    // target
	Method    string    // target method
	Sequence  int       // sequence number
}

func (m TellRequest) requestID() string   { return m.RequestID }
func (m TellRequest) value() []byte       { return m.Value }
func (m TellRequest) deadline() time.Time { return m.Deadline }
func (m TellRequest) target() Target      { return m.Target }
func (m TellRequest) method() string      { return m.Method }
func (m TellRequest) sequence() int       { return m.Sequence }

type Response struct {
	RequestID string    // request id
	Value     []byte    // payload
	Deadline  time.Time // request deadline
	ErrMsg    string    // error message or ""
	Node      string    // target node
}

func (m Response) requestID() string   { return m.RequestID }
func (m Response) value() []byte       { return m.Value }
func (m Response) deadline() time.Time { return m.Deadline }

type Done struct {
	RequestID string    // request id
	Deadline  time.Time // request deadline
}

func (m Done) requestID() string   { return m.RequestID }
func (m Done) value() []byte       { return nil } // nil value
func (m Done) deadline() time.Time { return m.Deadline }

func encode(topic string, partition int32, msg Message) *sarama.ProducerMessage {
	var meta map[string]string
	switch m := msg.(type) {
	case CallRequest:
		if m.Caller == "" {
			m.Caller = self.Node
		}
		meta = map[string]string{"Type": "Call", "RequestID": m.RequestID, "Method": m.Method, "Caller": m.Caller}
		if m.Sequence != 0 {
			meta["Sequence"] = strconv.Itoa(m.Sequence)
		}
		encodeTarget(m.Target, meta)
	case TellRequest:
		meta = map[string]string{"Type": "Tell", "RequestID": m.RequestID, "Method": m.Method}
		if m.Sequence != 0 {
			meta["Sequence"] = strconv.Itoa(m.Sequence)
		}
		encodeTarget(m.Target, meta)
	case Response:
		meta = map[string]string{"Type": "Response", "RequestID": m.RequestID, "ErrMsg": m.ErrMsg}
	case Done:
		meta = map[string]string{"Type": "Done", "RequestID": m.RequestID}
	}
	if !msg.deadline().IsZero() {
		meta["Deadline"] = strconv.FormatInt(msg.deadline().Unix(), 10)
	}
	headers := make([]sarama.RecordHeader, len(meta))
	i := 0
	for k, v := range meta {
		headers[i] = sarama.RecordHeader{Key: []byte(k), Value: []byte(v)}
		i++
	}
	return &sarama.ProducerMessage{
		Topic:     topic,
		Partition: partition,
		Headers:   headers,
		Value:     sarama.ByteEncoder(msg.value()),
	}
}

func decode(msg *sarama.ConsumerMessage) Message {
	meta := map[string]string{}
	for _, h := range msg.Headers {
		meta[string(h.Key)] = string(h.Value)
	}
	var deadline time.Time
	if d, ok := meta["Deadline"]; ok {
		u, _ := strconv.ParseInt(d, 10, 64)
		deadline = time.Unix(u, 0)
	}
	var sequence int
	if s, ok := meta["Sequence"]; ok {
		v, _ := strconv.Atoi(s)
		sequence = v
	}
	switch meta["Type"] {
	case "Call":
		return CallRequest{RequestID: meta["RequestID"], Sequence: sequence, Deadline: deadline, Target: decodeTarget(meta), Method: meta["Method"], Caller: meta["Caller"], Value: msg.Value}
	case "Tell":
		return TellRequest{RequestID: meta["RequestID"], Sequence: sequence, Deadline: deadline, Target: decodeTarget(meta), Method: meta["Method"], Value: msg.Value}
	case "Response":
		return Response{RequestID: meta["RequestID"], Deadline: deadline, ErrMsg: meta["ErrMsg"], Value: msg.Value}
	}
	return Done{RequestID: meta["RequestID"], Deadline: deadline}
}

func encodeTarget(target Target, meta map[string]string) {
	switch t := target.(type) {
	case Session:
		meta["Service"] = t.Name
		meta["Session"] = t.ID
	case Service:
		meta["Service"] = t.Name
	case Node:
		meta["Node"] = t.ID
	}
}

func decodeTarget(meta map[string]string) Target {
	if session, ok := meta["Session"]; ok {
		return Session{Name: meta["Service"], ID: session}
	} else if service, ok1 := meta["Service"]; ok1 {
		return Service{Name: service}
	}
	return Node{ID: meta["Node"]}
}
