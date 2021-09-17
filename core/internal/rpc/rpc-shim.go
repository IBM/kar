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

type karCallbackInfo struct {
	SendingNode string
	Request     string
}

type karMsg struct {
	Target   KarMsgTarget
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
func send(ctx context.Context, target KarMsgTarget, method string, callback karCallbackInfo, value []byte) error {
	select { // make sure we have joined
	case <-pubsub.Joined:
	case <-ctx.Done():
		return ctx.Err()
	}
	var partition int32
	var err error
	switch target.Protocol {
	case "service": // route to service
		partition, target.Node, err = pubsub.RouteToService(ctx, target.Name)
		if err != nil {
			logger.Error("failed to route to service %s: %v", target.Name, err)
			return err
		}
	case "actor": // route to actor
		partition, target.Node, err = pubsub.RouteToActor(ctx, target.Name, target.ID)
		if err != nil {
			logger.Error("failed to route to actor type %s, id %s: %v", target.Name, target.ID, err)
			return err
		}
	case "sidecar": // route to sidecar
		partition, err = pubsub.RouteToSidecar(target.Node)
		if err != nil {
			logger.Error("failed to route to sidecar %s: %v", target.Node, err)
			return err
		}
	case "partition": // route to partition
		partition = target.Partition
	}
	m, err := json.Marshal(karMsg{Target: target, Method: method, Callback: callback, Body: value})
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
	switch msg.Target.Protocol {
	case "service":
		if msg.Target.Name != config.ServiceName {
			forwarded = true
			err = TellKAR(ctx, msg.Target, msg.Method, msg.Body)
		}
	case "sidecar":
		if msg.Target.Node != GetNodeID() {
			forwarded = true
			err = forwardToSidecar(ctx, msg.Target, msg.Method, msg.Body)
		}
	}

	// If not forwarded elsewhere, actually dispatch up to the handler
	if !forwarded {
		if handler, ok := handlers[msg.Method]; ok {
			reply, err = handler(ctx, msg.Target, msg.Body)
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

func forwardToSidecar(ctx context.Context, target KarMsgTarget, method string, value []byte) error {
	err := TellKAR(ctx, target, method, value)
	if err == pubsub.ErrUnknownSidecar {
		logger.Debug("dropping message to dead sidecar %s: %v", target.Node, err)
		return nil
	}
	return err
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

	err = TellKAR(ctx, KarMsgTarget{Protocol: "sidecar", Node: callback.SendingNode}, responseMethod, value)

	if err == pubsub.ErrUnknownSidecar {
		logger.Debug("dropping answer to request %s from dead sidecar %s: %v", callback.Request, callback.SendingNode, err)
		return nil
	}
	return err
}

func responseHandler(ctx context.Context, target KarMsgTarget, value []byte) (*Reply, error) {
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
