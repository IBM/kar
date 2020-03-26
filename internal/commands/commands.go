package commands

import (
	"context"
	"net/http"
	"strconv"
	"sync"

	"github.com/google/uuid"
	"github.ibm.com/solsa/kar.git/internal/actors"
	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/internal/proxy"
	"github.ibm.com/solsa/kar.git/internal/pubsub"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

// pending requests: map request uuid (string) to channel (chan Reply)
var requests = sync.Map{}

// TellService sends a message to a service and does not wait for a reply
func TellService(ctx context.Context, service, path, payload, contentType string) error {
	return pubsub.Send(ctx, map[string]string{
		"protocol":     "service",
		"service":      service,
		"command":      "tell", // post with no callback expected
		"path":         path,
		"content-type": contentType,
		"payload":      payload})
}

// TellActor sends a message to an actor and does not wait for a reply
func TellActor(ctx context.Context, actor actors.Actor, path, payload, contentType string) error {
	return pubsub.Send(ctx, map[string]string{
		"protocol":     "actor",
		"type":         actor.Type,
		"id":           actor.ID,
		"command":      "tell", // post with no callback expected
		"path":         path,
		"content-type": contentType,
		"payload":      payload})
}

// Reply represents the return value of a call
type Reply struct {
	StatusCode  int
	ContentType string
	Payload     string
}

// CallService calls a service and waits for a reply
func CallService(ctx context.Context, service, path, payload, contentType, accept string) (*Reply, error) {
	request := uuid.New().String()
	ch := make(chan Reply)
	requests.Store(request, ch)
	defer requests.Delete(request)
	err := pubsub.Send(ctx, map[string]string{
		"protocol":     "service",
		"service":      service,
		"command":      "call",
		"path":         path,
		"content-type": contentType,
		"accept":       accept,
		"from":         config.ID, // this sidecar
		"request":      request,
		"payload":      payload})
	if err != nil {
		return nil, err
	}
	select {
	case r := <-ch:
		return &r, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// CallActor calls an actor and waits for a reply
func CallActor(ctx context.Context, actor actors.Actor, path, payload, contentType, accept string) (*Reply, error) {
	request := uuid.New().String()
	ch := make(chan Reply)
	requests.Store(request, ch)
	defer requests.Delete(request)
	err := pubsub.Send(ctx, map[string]string{
		"protocol":     "actor",
		"type":         actor.Type,
		"id":           actor.ID,
		"command":      "call",
		"path":         path,
		"content-type": contentType,
		"accept":       accept,
		"from":         config.ID, // this sidecar
		"request":      request,
		"payload":      payload})
	if err != nil {
		return nil, err
	}
	select {
	case r := <-ch:
		return &r, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Broadcast sends a message to all sidecars except for this one
func Broadcast(ctx context.Context, path, payload, contentType string) {
	for _, sidecar := range pubsub.Sidecars() {
		if sidecar != config.ID { // send to all other sidecars
			pubsub.Send(ctx, map[string]string{ // TODO log errors
				"protocol":     "sidecar",
				"sidecar":      sidecar,
				"command":      "tell",
				"path":         path,
				"content-type": contentType,
				"payload":      payload})
		}
	}
}

// KillAll sends the cancel command to all sidecars except for this one
func KillAll(ctx context.Context) {
	for _, sidecar := range pubsub.Sidecars() {
		if sidecar != config.ID { // send to all other sidecars
			pubsub.Send(ctx, map[string]string{ // TODO log errors
				"protocol": "sidecar",
				"sidecar":  sidecar,
				"command":  "cancel",
			})
		}
	}
}

// helper methods to handle incoming messages
// return either nil or ctx.Err() if cancelled
// log ignored errors to logger.Error

func tell(ctx context.Context, msg map[string]string) error {
	res, err := proxy.Do(ctx, "POST", msg)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		logger.Error("failed to post to %s: %v", msg["path"], err)
	} else {
		proxy.Flush(res.Body)
	}
	return nil
}

func call(ctx context.Context, msg map[string]string) error {
	var reply Reply
	res, err := proxy.Do(ctx, "POST", msg)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		logger.Warning("failed to post to %s: %v", msg["path"], err) // return error to caller
		reply = Reply{StatusCode: http.StatusBadGateway, Payload: "Bad Gateway", ContentType: "text/plain"}
	} else {
		reply = Reply{StatusCode: res.StatusCode, Payload: proxy.Read(res.Body), ContentType: res.Header.Get("Content-Type")}
	}
	err = pubsub.Send(ctx, map[string]string{
		"protocol":     "sidecar",
		"sidecar":      msg["from"],
		"command":      "callback",
		"request":      msg["request"],
		"statusCode":   strconv.Itoa(reply.StatusCode),
		"content-type": reply.ContentType,
		"payload":      reply.Payload})
	if err != nil {
		// TODO distinguish dead sidecar from other errors
		logger.Debug("failed to answer request %s from sidecar %s: %v", msg["request"], msg["from"], err)
	}
	return nil
}

func callback(ctx context.Context, msg map[string]string) error {
	if ch, ok := requests.Load(msg["request"]); ok {
		statusCode, _ := strconv.Atoi(msg["statusCode"])
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ch.(chan Reply) <- Reply{StatusCode: statusCode, ContentType: msg["content-type"], Payload: msg["payload"]}:
		}
	} else {
		logger.Error("unexpected request in callback %s", msg["request"])
	}
	return nil
}

func dispatch(ctx context.Context, cancel context.CancelFunc, msg map[string]string) error {
	switch msg["command"] {
	case "tell":
		return tell(ctx, msg)
	case "call":
		return call(ctx, msg)
	case "callback":
		return callback(ctx, msg)
	case "cancel":
		cancel() // never fails
	default:
		logger.Error("unexpected command %s", msg["command"])
	}
	return nil
}

func forwardToService(ctx context.Context, msg map[string]string) error {
	if err := pubsub.Send(ctx, msg); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		logger.Error("failed to forward message to service %s: %v", msg["service"], err)
	}
	return nil
}
func forwardToActor(ctx context.Context, msg map[string]string) error {
	if err := pubsub.Send(ctx, msg); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		logger.Error("failed to forward message to actor %v: %v", actors.Actor{Type: msg["type"], ID: msg["id"]}, err)
	}
	return nil
}

func forwardToSidecar(ctx context.Context, msg map[string]string) error {
	if err := pubsub.Send(ctx, msg); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// TODO distinguish dead sidecar from other errors
		logger.Debug("failed to forward message to sidecar %s: %v", msg["sidecar"], err)
	}
	return nil
}

// Process processes one incoming message
func Process(ctx context.Context, cancel context.CancelFunc, message pubsub.Message) {
	msg := message.Value
	var err error
	switch msg["protocol"] {
	case "service":
		if msg["service"] == config.ServiceName {
			err = dispatch(ctx, cancel, msg)
		} else {
			err = forwardToService(ctx, msg)
		}
	case "sidecar":
		if msg["sidecar"] == config.ID {
			err = dispatch(ctx, cancel, msg)
		} else {
			err = forwardToSidecar(ctx, msg)
		}
	case "actor":
		actor := actors.Actor{Type: msg["type"], ID: msg["id"]}
		e, fresh, _ := actors.Acquire(ctx, actor)
		if e == nil && ctx.Err() == nil {
			err = forwardToActor(ctx, msg)
		} else {
			defer e.Release()
			if fresh {
				Activate(ctx, actor)
			}
			msg["path"] = "/actor/" + actor.Type + "/" + actor.ID + msg["path"]
			err = dispatch(ctx, cancel, msg)
		}
	}
	if err == nil {
		message.Mark()
	}
}

// actors

// Activate activates an actor
func Activate(ctx context.Context, actor actors.Actor) {
	logger.Debug("activating actor %v", actor)
	res, err := proxy.Do(ctx, "GET", map[string]string{"path": "/actor/" + actor.Type + "/" + actor.ID})
	if err != nil {
		logger.Error("failed to activate actor %v: %v", actor, err)
	} else if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNotFound {
		logger.Error("failed to activate actor %v: %s", actor, proxy.Read(res.Body))
	} else {
		proxy.Flush(res.Body)
	}
}

// Deactivate deactivates an actor (but retains placement)
func Deactivate(ctx context.Context, actor actors.Actor) {
	logger.Debug("deactivating actor %v", actor)
	res, err := proxy.Do(ctx, "DELETE", map[string]string{"path": "/actor/" + actor.Type + "/" + actor.ID})
	if err != nil {
		logger.Error("failed to deactivate actor %v: %v", actor, err)
	} else if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNotFound {
		logger.Error("failed to deactivate actor %v: %s", actor, proxy.Read(res.Body))
	} else {
		proxy.Flush(res.Body)
	}
}

// Migrate deactivates an actor if active and resets its placement
func Migrate(ctx context.Context, actor actors.Actor) {
	actors.Migrate(ctx, actor, Deactivate)
}
