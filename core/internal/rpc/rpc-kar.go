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

// TellService sends a message to a service and does not wait for a reply
func TellService(ctx context.Context, service Service, path, payload, header, method string) error {
	msg := map[string]string{
		"command": "tell", // post with no callback expected
		"path":    path,
		"header":  header,
		"method":  method,
		"payload": payload}

	return pubsub.Send(ctx, pubsub.KarStructuredMsg{
		Protocol: "service",
		Name:     service.Name,
		Msg:      msg})
}

// TellActor sends a message to an actor and does not wait for a reply
func TellActor(ctx context.Context, actor Session, path, payload string) error {
	msg := map[string]string{
		"command": "tell", // post with no callback expected
		"path":    path,
		"payload": payload}

	return pubsub.Send(ctx, pubsub.KarStructuredMsg{
		Protocol: "actor",
		Name:     actor.Name,
		ID:       actor.ID,
		Msg:      msg})
}

// DeleteActor sends a delete message to an actor and does not wait for a reply
func DeleteActor(ctx context.Context, actor Session) error {
	msg := map[string]string{"command": "delete"}

	return pubsub.Send(ctx, pubsub.KarStructuredMsg{
		Protocol: "actor",
		Name:     actor.Name,
		ID:       actor.ID,
		Msg:      msg})
}

func TellBinding(ctx context.Context, kind string, actor Session, partition int32, bindingID string) error {
	msg := map[string]string{
		"command":   "binding:tell",
		"kind":      kind,
		"bindingId": bindingID}

	return pubsub.Send(ctx, pubsub.KarStructuredMsg{
		Protocol:  "actor",
		Name:      actor.Name,
		ID:        actor.ID,
		Partition: partition,
		Msg:       msg})
}

// CallSidecar makes a call via pubsub to a sidecar and waits for a reply
func CallSidecar(ctx context.Context, msg pubsub.KarStructuredMsg) (*Reply, error) {
	return callHelper(ctx, msg)
}

// callHelper makes a call via pubsub to a sidecar and waits for a reply
func callHelper(ctx context.Context, msg pubsub.KarStructuredMsg) (*Reply, error) {
	request := uuid.New().String()
	ch := make(chan *Reply)
	requests.Store(request, ch)
	defer requests.Delete(request)
	msg.Msg["from"] = config.ID // this sidecar
	msg.Msg["request"] = request
	err := pubsub.Send(ctx, msg)
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
func CallPromiseSidecar(ctx context.Context, msg pubsub.KarStructuredMsg) (string, error) {
	return callPromiseHelper(ctx, msg)
}

func callPromiseHelper(ctx context.Context, msg pubsub.KarStructuredMsg) (string, error) {
	request := uuid.New().String()
	ch := make(chan *Reply)
	requests.Store(request, ch)
	// defer requests.Delete(request)
	msg.Msg["from"] = config.ID // this sidecar
	msg.Msg["request"] = request
	err := pubsub.Send(ctx, msg)
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
func Callback(ctx context.Context, msg pubsub.KarStructuredMsg) error {
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
