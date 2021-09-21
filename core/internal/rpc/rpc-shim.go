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

////////////////////////////////////////////////////
// pubsub ("legacy") based implementation of rpc library API
////////////////////////////////////////////////////

package rpc

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/IBM/kar/core/internal/config"
	"github.com/IBM/kar/core/internal/pubsub"
	"github.com/IBM/kar/core/pkg/logger"
	"github.com/google/uuid"
)

var (
	// pending requests: map request uuid (string) to channel (chan Result)
	requests = sync.Map{}
	handlers = make(map[string]Handler)
)

const (
	responseMethod = "response"
)

type publisher struct {
}

type karCallbackInfo struct {
	SendingNode string
	Request     string
}

type karTarget struct {
	Type string
	Name string
	ID   string
}

type karMsg struct {
	Target   karTarget
	Sidecar  string
	Callback karCallbackInfo
	Method   string
	Body     []byte
}

func init() {
	register(responseMethod, responseHandler)
}

func register(method string, handler Handler) {
	handlers[method] = handler
}

func connect(ctx context.Context, conf *Config, services ...string) (<-chan struct{}, error) {
	logger.Fatal("Unimplemented rpc-shim function")
	return nil, nil
}

func call(ctx context.Context, target Target, method string, value []byte) ([]byte, error) {
	request := uuid.New().String()
	ch := make(chan Result)
	requests.Store(request, ch)
	defer requests.Delete(request)
	err := send(ctx, target, method, karCallbackInfo{SendingNode: getNodeID(), Request: request}, value)
	if err != nil {
		return nil, err
	}
	select {
	case r := <-ch:
		return r.Value, r.Err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func tell(ctx context.Context, target Target, method string, value []byte) error {
	return send(ctx, target, method, karCallbackInfo{}, value)
}

func async(ctx context.Context, target Target, method string, value []byte) (string, <-chan Result, error) {
	request := uuid.New().String()
	ch := make(chan Result)
	requests.Store(request, ch)
	err := send(ctx, target, method, karCallbackInfo{SendingNode: getNodeID(), Request: request}, value)
	if err != nil {
		return "", nil, err
	}
	return request, ch, nil
}

func reclaim(requestID string) {
	requests.Delete(requestID)
}

func getServices() ([]string, <-chan struct{}) {
	logger.Fatal("Unimplemented rpc-shim function")
	return nil, nil
}

func getNodeID() string {
	return config.ID
}

func getNodeIDs() ([]string, <-chan struct{}) {
	return pubsub.Sidecars(), nil // TODO: Kar doesn't use the notification channel, so not bothering to implement it
}

func getServiceNodeIDs(service string) ([]string, <-chan struct{}) {
	logger.Fatal("Unimplemented rpc-shim function")
	return nil, nil
}

func getPartition() int32 {
	logger.Fatal("Unimplemented rpc-shim function")
	return 0
}

func getSessionNodeID(ctx context.Context, session Session) (string, error) {
	return pubsub.GetSidecar(ctx, session.Name, session.ID)
}

func getPartitions() ([]int32, <-chan struct{}) {
	return pubsub.Partitions()
}

func delSession(ctx context.Context, session Session) error {
	_, err := pubsub.CompareAndSetSidecar(ctx, session.Name, session.ID, getNodeID(), "")
	return err
}

func createTopic(conf *Config, topic string, parameters string) error {
	logger.Fatal("Unimplemented rpc-shim function")
	return nil
}

func deleteTopic(conf *Config, topic string) error {
	logger.Fatal("Unimplemented rpc-shim function")
	return nil
}

func newPublisher(conf *Config) (*Publisher, error) {
	logger.Fatal("Unimplemented rpc-shim function")
	return nil, nil
}

func (p *Publisher) publish(topic string, value []byte) error {
	logger.Fatal("Unimplemented rpc-shim function")
	return nil
}

func (p *Publisher) close() error {
	logger.Fatal("Unimplemented rpc-shim function")
	return nil
}

func subscribe(ctx context.Context, conf *Config, topic, group string, oldest bool, handler func([]byte)) error {
	logger.Fatal("Unimplemented rpc-shim function")
	return nil
}

////
// lowlevel request support in caller
////

// send sends message to receiver
func send(ctx context.Context, target Target, method string, callback karCallbackInfo, value []byte) error {
	select { // make sure we have joined
	case <-pubsub.Joined:
	case <-ctx.Done():
		return ctx.Err()
	}
	var kt karTarget
	var partition int32
	var sidecar string
	var err error
	switch t := target.(type) {
	case Service: // route to service
		partition, sidecar, err = pubsub.RouteToService(ctx, t.Name)
		kt = karTarget{Type: "service", Name: t.Name}
		if err != nil {
			logger.Error("failed to route to service %s: %v", t.Name, err)
			return err
		}
	case Session: // route to actor
		partition, sidecar, err = pubsub.RouteToActor(ctx, t.Name, t.ID)
		kt = karTarget{Type: "session", Name: t.Name, ID: t.ID}
		if err != nil {
			logger.Error("failed to route to actor type %s, id %s: %v", t.Name, t.ID, err)
			return err
		}
	case Node: // route to sidecar
		partition, err = pubsub.RouteToSidecar(t.ID)
		sidecar = t.ID
		kt = karTarget{Type: "node", ID: t.ID}
		if err != nil {
			logger.Error("failed to route to sidecar %s: %v", t.ID, err)
			return err
		}
	}
	m, err := json.Marshal(karMsg{Target: kt, Sidecar: sidecar, Method: method, Callback: callback, Body: value})
	if err != nil {
		logger.Error("failed to marshal message: %v", err)
		return err
	}
	return pubsub.SendBytes(ctx, partition, m)
}

////
// lowlevel request support in callee
////

// Process processes one incoming message
func Process(ctx context.Context, cancel context.CancelFunc, message pubsub.Message) {
	var msg karMsg
	err := json.Unmarshal(message.Value, &msg)
	if err != nil {
		logger.Error("failed to unmarshal message: %v", err)
		message.Mark()
		return
	}
	var target Target
	if msg.Target.Type == "service" {
		target = Service{Name: msg.Target.Name}
	} else if msg.Target.Type == "session" {
		target = Session{Name: msg.Target.Name, ID: msg.Target.ID}
	} else if msg.Target.Type == "node" {
		target = Node{ID: msg.Target.ID}
	} else {
		logger.Error("unknown message target type %v", msg.Target.Type)
		message.Mark()
		return
	}

	// Cancellation of Calls from dead nodes
	if msg.Callback.Request != "" && !pubsub.IsLiveSidecar(msg.Callback.SendingNode) {
		logger.Info("Cancelling request %v from dead sidecar %s", msg.Callback.Request, msg.Callback.SendingNode)
		message.Mark()
		return
	}

	// Forwarding
	forwarded := false
	switch t := target.(type) {
	case Service:
		if t.Name != config.ServiceName {
			forwarded = true
			err = Tell(ctx, target, msg.Method, msg.Body)
		}
	case Node:
		if t.ID != GetNodeID() {
			forwarded = true
			err := Tell(ctx, target, msg.Method, msg.Body)
			if err == pubsub.ErrUnknownSidecar {
				logger.Debug("dropping message to dead sidecar %s: %v", t.ID, err)
				err = nil
			}
		}
	}

	// If not forwarded elsewhere, actually dispatch up to the handler
	if !forwarded {
		if handler, ok := handlers[msg.Method]; ok {
			reply, err := handler(ctx, target, msg.Body)
			if err == nil && reply != nil {
				err = respond(ctx, msg.Callback, reply)
			}
		} else {
			logger.Error("Dropping message for unknown handler %v", msg.Method)
		}
	}

	if err == nil {
		message.Mark()
	}
}

////
// lowlevel reponse support in caller
////

type callResponse struct {
	Request string
	Value   []byte
}

func respond(ctx context.Context, callback karCallbackInfo, reply []byte) error {
	response := callResponse{Request: callback.Request, Value: reply}
	value, err := json.Marshal(response)
	if err != nil {
		logger.Error("respond: failed to serialize response: %v", err)
		return err
	}

	err = Tell(ctx, Node{ID: callback.SendingNode}, responseMethod, value)

	if err == pubsub.ErrUnknownSidecar {
		logger.Debug("dropping answer to request %s from dead sidecar %s: %v", callback.Request, callback.SendingNode, err)
		return nil
	}
	return err
}

func responseHandler(ctx context.Context, target Target, value []byte) ([]byte, error) {
	var response callResponse
	err := json.Unmarshal(value, &response)
	if err != nil {
		logger.Error("responseHandler: failed to unmarshal response: %v", err)
		return nil, err
	}

	if ch, ok := requests.Load(response.Request); ok {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case ch.(chan Result) <- Result{Value: response.Value, Err: nil}:
		}
	} else {
		logger.Error("unexpected request in callback %s", response.Request)
	}
	return nil, nil
}
