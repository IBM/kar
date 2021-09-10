package rpc

import (
	"context"
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

// Reply represents the return value of a call
type Reply struct {
	StatusCode  int
	ContentType string
	Payload     string
}

type ActorTarget struct {
	Type string // actor type
	ID   string // actor instance id
}

// TellService sends a message to a service and does not wait for a reply
func TellService(ctx context.Context, service, path, payload, header, method string, direct bool) error {
	return pubsub.Send(ctx, direct, map[string]string{
		"protocol": "service",
		"service":  service,
		"command":  "tell", // post with no callback expected
		"path":     path,
		"header":   header,
		"method":   method,
		"payload":  payload})
}

// TellActor sends a message to an actor and does not wait for a reply
func TellActor(ctx context.Context, actor ActorTarget, path, payload string, direct bool) error {
	return pubsub.Send(ctx, direct, map[string]string{
		"protocol": "actor",
		"type":     actor.Type,
		"id":       actor.ID,
		"command":  "tell", // post with no callback expected
		"path":     path,
		"payload":  payload})
}

// DeleteActor sends a delete message to an actor and does not wait for a reply
func DeleteActor(ctx context.Context, actor ActorTarget, direct bool) error {
	return pubsub.Send(ctx, direct, map[string]string{
		"protocol": "actor",
		"type":     actor.Type,
		"id":       actor.ID,
		"command":  "delete"})
}

func TellBinding(ctx context.Context, kind string, actor ActorTarget, partition, bindingID string) error {
	return pubsub.Send(ctx, false, map[string]string{
		"protocol":  "actor",
		"type":      actor.Type,
		"id":        actor.ID,
		"command":   "binding:tell",
		"kind":      kind,
		"partition": partition,
		"bindingId": bindingID})
}

// CallSidecar makes a call via pubsub to a sidecar and waits for a reply
func CallSidecar(ctx context.Context, msg map[string]string, direct bool) (*Reply, error) {
	return callHelper(ctx, msg, direct)
}

// callHelper makes a call via pubsub to a sidecar and waits for a reply
func callHelper(ctx context.Context, msg map[string]string, direct bool) (*Reply, error) {
	request := uuid.New().String()
	ch := make(chan *Reply)
	requests.Store(request, ch)
	defer requests.Delete(request)
	msg["from"] = config.ID // this sidecar
	msg["request"] = request
	err := pubsub.Send(ctx, direct, msg)
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
func CallPromiseSidecar(ctx context.Context, msg map[string]string, direct bool) (string, error) {
	return callPromiseHelper(ctx, msg, direct)
}

func callPromiseHelper(ctx context.Context, msg map[string]string, direct bool) (string, error) {
	request := uuid.New().String()
	ch := make(chan *Reply)
	requests.Store(request, ch)
	// defer requests.Delete(request)
	msg["from"] = config.ID // this sidecar
	msg["request"] = request
	err := pubsub.Send(ctx, direct, msg)
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
func Callback(ctx context.Context, msg map[string]string) error {
	if ch, ok := requests.Load(msg["request"]); ok {
		statusCode, _ := strconv.Atoi(msg["statusCode"])
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ch.(chan *Reply) <- &Reply{StatusCode: statusCode, ContentType: msg["content-type"], Payload: msg["payload"]}:
		}
	} else {
		logger.Error("unexpected request in callback %s", msg["request"])
	}
	return nil
}
