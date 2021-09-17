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
)

var (
	// pending requests: map request uuid (string) to channel (chan Reply)
	requests = sync.Map{}
	handlers = make(map[string]KarHandler)
)

const (
	responseMethod = "response"
)

// Reply represents the return value of a call  TODO: Move back to runtime (commands.go)
type Reply struct {
	StatusCode  int
	ContentType string
	Payload     string
}

func (s Service) target() {}
func (s Session) target() {}
func (s Node) target()    {}

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
	RegisterKAR(responseMethod, responseHandler)
}

// Register method handler
func register(method string, handler Handler) {

}

// Connect to Kafka
func connect(ctx context.Context, conf *Config, services ...string) (<-chan struct{}, error) {
	logger.Fatal("Unimplemented rpc-shim function")
	return nil, nil
}

// Call method and wait for result
func call(ctx context.Context, target Target, method string, value []byte) ([]byte, error) {
	logger.Fatal("Unimplemented rpc-shim function")
	return nil, nil
}

// Call method and return immediately (result will be discarded)
func tell(ctx context.Context, target Target, method string, value []byte) error {
	logger.Fatal("Unimplemented rpc-shim function")
	return nil
}

// Call method and return a request id and a result channel
func async(ctx context.Context, target Target, method string, value []byte) (string, <-chan Result, error) {
	logger.Fatal("Unimplemented rpc-shim function")
	return "", nil, nil
}

// Reclaim resources associated with async request id
func reclaim(requestID string) {
	logger.Fatal("Unimplemented rpc-shim function")
}

func getServices() ([]string, <-chan struct{}) {
	logger.Fatal("Unimplemented rpc-shim function")
	return nil, nil
}

// GetNodeID returns the node id for the current node
func getNodeID() string {
	return config.ID
}

// GetNodeIDs returns the sorted list of live node ids and a channel to be notified of changes
func getNodeIDs() ([]string, <-chan struct{}) {
	return pubsub.Sidecars(), nil
}

// GetServiceNodeIDs returns the sorted list of live node ids for a given service
func getServiceNodeIDs(service string) []string {
	logger.Fatal("Unimplemented rpc-shim function")
	return nil
}

// GetPartition returns the partition for the current node
func getPartition() int32 {
	logger.Fatal("Unimplemented rpc-shim function")
	return 0
}

// GetPartitions returns the sorted list of partitions in use
func getPartitions() ([]int32, <-chan struct{}) {
	logger.Fatal("Unimplemented rpc-shim function")
	return nil, nil
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
	var reply *Reply = nil
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
	/*
		 * TODO: restore this functionality
		         Need to cancel calls (but not tells) that originated from dead sidecars
		if !pubsub.IsLiveSidecar(msg.Msg["from"]) {
			logger.Info("Cancelling %s from dead sidecar %s", msg.Msg["method"], msg.Msg["from"])
			return nil, nil
		}
	*/

	// Forwarding
	forwarded := false
	switch t := target.(type) {
	case Service:
		if t.Name != config.ServiceName {
			forwarded = true
			err = TellKAR(ctx, target, msg.Method, msg.Body)
		}
	case Node:
		if t.ID != GetNodeID() {
			forwarded = true
			err := TellKAR(ctx, target, msg.Method, msg.Body)
			if err == pubsub.ErrUnknownSidecar {
				logger.Debug("dropping message to dead sidecar %s: %v", t.ID, err)
				err = nil
			}
		}
	}

	// If not forwarded elsewhere, actually dispatch up to the handler
	if !forwarded {
		if handler, ok := handlers[msg.Method]; ok {
			reply, err = handler(ctx, target, msg.Body)
			if reply != nil {
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
	Value   Reply
}

func respond(ctx context.Context, callback karCallbackInfo, reply *Reply) error {
	response := callResponse{Request: callback.Request, Value: *reply}
	value, err := json.Marshal(response)
	if err != nil {
		logger.Error("respond: failed to serialize response: %v", err)
		return err
	}

	err = TellKAR(ctx, Node{ID: callback.SendingNode}, responseMethod, value)

	if err == pubsub.ErrUnknownSidecar {
		logger.Debug("dropping answer to request %s from dead sidecar %s: %v", callback.Request, callback.SendingNode, err)
		return nil
	}
	return err
}

func responseHandler(ctx context.Context, target Target, value []byte) (*Reply, error) {
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
		case ch.(chan *Reply) <- &response.Value:
		}
	} else {
		logger.Error("unexpected request in callback %s", response.Request)
	}
	return nil, nil
}
