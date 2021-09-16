package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/IBM/kar/core/internal/config"
	"github.com/IBM/kar/core/internal/pubsub"
	"github.com/IBM/kar/core/pkg/logger"
	"github.com/google/uuid"
)

var (
	// pending requests: map request uuid (string) to channel (chan Reply)
	requests = sync.Map{}
	handlers = make(map[string]KarHandler)
)

const (
	responseMethod = "response"
)

// Staging types to allow migration to new RPC library
type KarMsgTarget struct {
	Protocol  string
	Name      string
	ID        string
	Node      string
	Partition int32
}

type KarHandler func(context.Context, KarMsgTarget, []byte) (*Reply, error)

// Reply represents the return value of a call
type Reply struct {
	StatusCode  int
	ContentType string
	Payload     string
}

// Internal types specific to the original pub-sub based implementation of the library

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

////////
// Staging code...these methods are meant to be directly replacable by their corresponding RPC versions once the APIs converge
////////

func RegisterKAR(method string, handler KarHandler) {
	handlers[method] = handler
}

// TellKAR makes a call via pubsub to a sidecar and returns immediately (result will be discarded)
func TellKAR(ctx context.Context, target KarMsgTarget, method string, value []byte) error {
	return send(ctx, target, method, karCallbackInfo{}, value)
}

// CallKAR makes a call via pubsub to a sidecar and waits for a reply
func CallKAR(ctx context.Context, target KarMsgTarget, method string, value []byte) (*Reply, error) {
	request := uuid.New().String()
	ch := make(chan *Reply)
	requests.Store(request, ch)
	defer requests.Delete(request)
	err := send(ctx, target, method, karCallbackInfo{SendingNode: getNodeID(), Request: request}, value)
	if err != nil {
		return nil, err
	}
	select {
	case r := <-ch:
		return r, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// CallPromiseKAR makes a call via pubsub to a sidecar and returns a promise that may later be used to await the reply
func CallPromiseKAR(ctx context.Context, target KarMsgTarget, method string, value []byte) (string, error) {
	request := uuid.New().String()
	ch := make(chan *Reply)
	requests.Store(request, ch)
	// defer requests.Delete(request)
	err := send(ctx, target, method, karCallbackInfo{SendingNode: getNodeID(), Request: request}, value)
	if err != nil {
		return "", err
	}
	return request, nil
}

// AwaitPromiseKAR awaits the response to an actor or service call
func AwaitPromiseKAR(ctx context.Context, request string) (*Reply, error) {
	if ch, ok := requests.Load(request); ok {
		defer requests.Delete(request)
		select {
		case r := <-ch.(chan *Reply):
			return r, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, fmt.Errorf("unexpected request %s", request)
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

	err = TellKAR(ctx,
		KarMsgTarget{Protocol: "sidecar", Node: callback.SendingNode},
		responseMethod,
		value)

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
