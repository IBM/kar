package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/IBM/kar/core/internal/config"
	"github.com/IBM/kar/core/internal/pubsub"
	"github.com/IBM/kar/core/pkg/logger"
	"github.com/google/uuid"
)

var (
	// pending requests: map request uuid (string) to channel (chan Reply)
	requests = sync.Map{}
)

// Staging types to allow migration to new RPC library
type KarMsgTarget struct {
	Protocol  string
	Name      string
	ID        string
	Node      string
	Partition int32
}

type KarMsgBody struct {
	Msg map[string]string
}

type KarMsg struct {
	Target KarMsgTarget
	Body   KarMsgBody
}

// Reply represents the return value of a call
type Reply struct {
	StatusCode  int
	ContentType string
	Payload     string
}

// TellService sends a message to a service and does not wait for a reply
func TellService(ctx context.Context, service Service, path, payload, header, method string) error {
	msg := map[string]string{
		"command": "tell", // post with no callback expected
		"path":    path,
		"header":  header,
		"method":  method,
		"payload": payload}

	return Send(ctx,
		KarMsgTarget{Protocol: "service", Name: service.Name},
		KarMsgBody{Msg: msg})
}

// TellActor sends a message to an actor and does not wait for a reply
func TellActor(ctx context.Context, actor Session, path, payload string) error {
	msg := map[string]string{
		"command": "tell", // post with no callback expected
		"path":    path,
		"payload": payload}

	return Send(ctx,
		KarMsgTarget{Protocol: "actor", Name: actor.Name, ID: actor.ID},
		KarMsgBody{Msg: msg})
}

// DeleteActor sends a delete message to an actor and does not wait for a reply
func DeleteActor(ctx context.Context, actor Session) error {
	msg := map[string]string{"command": "delete"}

	return Send(ctx,
		KarMsgTarget{Protocol: "actor", Name: actor.Name, ID: actor.ID},
		KarMsgBody{Msg: msg})
}

func TellBinding(ctx context.Context, kind string, actor Session, partition int32, bindingID string) error {
	msg := map[string]string{
		"command":   "binding:tell",
		"kind":      kind,
		"bindingId": bindingID}

	return Send(ctx,
		KarMsgTarget{Protocol: "actor", Name: actor.Name, ID: actor.ID, Partition: partition},
		KarMsgBody{Msg: msg})
}

// CallSidecar makes a call via pubsub to a sidecar and waits for a reply
func CallSidecar(ctx context.Context, target KarMsgTarget, msg KarMsgBody) (*Reply, error) {
	return callHelper(ctx, target, msg)
}

// callHelper makes a call via pubsub to a sidecar and waits for a reply
func callHelper(ctx context.Context, target KarMsgTarget, msg KarMsgBody) (*Reply, error) {
	request := uuid.New().String()
	ch := make(chan *Reply)
	requests.Store(request, ch)
	defer requests.Delete(request)
	msg.Msg["from"] = config.ID // this sidecar
	msg.Msg["request"] = request
	err := Send(ctx, target, msg)
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

// CallPromiseSidecar makes a call via pubsub to a sidecar and returns a promise that may later be used to await the reply
func CallPromiseSidecar(ctx context.Context, target KarMsgTarget, msg KarMsgBody) (string, error) {
	return callPromiseHelper(ctx, target, msg)
}

func callPromiseHelper(ctx context.Context, target KarMsgTarget, msg KarMsgBody) (string, error) {
	request := uuid.New().String()
	ch := make(chan *Reply)
	requests.Store(request, ch)
	// defer requests.Delete(request)
	msg.Msg["from"] = config.ID // this sidecar
	msg.Msg["request"] = request
	err := Send(ctx, target, msg)
	if err != nil {
		return "", err
	}
	return request, nil
}

// AwaitPromise awaits the response to an actor or service call
func AwaitPromise(ctx context.Context, request string) (*Reply, error) {
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

// TEMP: This should not be exposed; temporary until i push handler registration down here.
func Callback(ctx context.Context, target KarMsgTarget, msg KarMsgBody) error {
	if ch, ok := requests.Load(msg.Msg["request"]); ok {
		statusCode, _ := strconv.Atoi(msg.Msg["statusCode"])
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ch.(chan *Reply) <- &Reply{StatusCode: statusCode, ContentType: msg.Msg["content-type"], Payload: msg.Msg["payload"]}:
		}
	} else {
		logger.Error("unexpected request in callback %s", msg.Msg["request"])
	}
	return nil
}

// Send sends message to receiver
func Send(ctx context.Context, target KarMsgTarget, msg KarMsgBody) error {
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
	m, err := json.Marshal(KarMsg{Target: target, Body: msg})
	if err != nil {
		logger.Error("failed to marshal message: %v", err)
		return err
	}
	return pubsub.SendBytes(ctx, partition, m)
}
