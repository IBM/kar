// Package runtime implements the core sidecar capabilities
package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.ibm.com/solsa/kar.git/internal/config"
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
func TellActor(ctx context.Context, actor Actor, path, payload, contentType string) error {
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

// callHelper makes a call via pubsub to a sidecar and waits for a reply
func callHelper(ctx context.Context, msg map[string]string) (*Reply, error) {
	request := uuid.New().String()
	ch := make(chan Reply)
	requests.Store(request, ch)
	defer requests.Delete(request)
	msg["from"] = config.ID // this sidecar
	msg["request"] = request
	err := pubsub.Send(ctx, msg)
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

// CallService calls a service and waits for a reply
func CallService(ctx context.Context, service, path, payload, contentType, accept string) (*Reply, error) {
	msg := map[string]string{
		"protocol":     "service",
		"service":      service,
		"command":      "call",
		"path":         path,
		"content-type": contentType,
		"accept":       accept,
		"payload":      payload}
	return callHelper(ctx, msg)
}

// CallActor calls an actor and waits for a reply
func CallActor(ctx context.Context, actor Actor, path, payload, contentType, accept string) (*Reply, error) {
	msg := map[string]string{
		"protocol":     "actor",
		"type":         actor.Type,
		"id":           actor.ID,
		"command":      "call",
		"path":         path,
		"content-type": contentType,
		"accept":       accept,
		"payload":      payload}
	return callHelper(ctx, msg)
}

// Reminders sends a reminder command (cancel, get, schedule) to an actor's sidecar and waits for a reply
func Reminders(ctx context.Context, actor Actor, action, payload, contentType, accept string) (*Reply, error) {
	msg := map[string]string{
		"protocol":     "actor",
		"type":         actor.Type,
		"id":           actor.ID,
		"command":      "reminder:" + action,
		"content-type": contentType,
		"accept":       accept,
		"payload":      payload}
	return callHelper(ctx, msg)
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

func respond(ctx context.Context, msg map[string]string, reply Reply) error {
	err := pubsub.Send(ctx, map[string]string{
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

func call(ctx context.Context, msg map[string]string) error {
	var reply Reply
	res, err := invoke(ctx, "POST", msg)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		logger.Warning("failed to post to %s: %v", msg["path"], err) // return error to caller
		reply = Reply{StatusCode: http.StatusBadGateway, Payload: "Bad Gateway", ContentType: "text/plain"}
	} else {
		reply = Reply{StatusCode: res.StatusCode, Payload: ReadAll(res.Body), ContentType: res.Header.Get("Content-Type")}
	}

	return respond(ctx, msg, reply)
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

func reminderCancel(ctx context.Context, msg map[string]string) error {
	var reply Reply
	actor := Actor{Type: msg["type"], ID: msg["id"]}
	found, err := CancelReminders(actor, msg["payload"], msg["content-type"], msg["accepts"])
	if err != nil {
		reply = Reply{StatusCode: http.StatusBadRequest, Payload: err.Error(), ContentType: "text/plain"}
	} else {
		reply = Reply{StatusCode: http.StatusOK, Payload: fmt.Sprintf("%v", found), ContentType: "text/plain"}
	}
	return respond(ctx, msg, reply)
}

func reminderGet(ctx context.Context, msg map[string]string) error {
	var reply Reply
	actor := Actor{Type: msg["type"], ID: msg["id"]}
	found, err := GetReminders(actor, msg["payload"], msg["content-type"], msg["accepts"])
	if err != nil {
		reply = Reply{StatusCode: http.StatusBadRequest, Payload: err.Error(), ContentType: "text/plain"}
	} else {
		blob, err := json.Marshal(found)
		if err != nil {
			reply = Reply{StatusCode: http.StatusInternalServerError, Payload: err.Error(), ContentType: "text/plain"}
		} else {
			reply = Reply{StatusCode: http.StatusOK, Payload: string(blob), ContentType: "application/json"}
		}
	}
	return respond(ctx, msg, reply)
}

func reminderSchedule(ctx context.Context, msg map[string]string) error {
	var reply Reply
	actor := Actor{Type: msg["type"], ID: msg["id"]}
	err := ScheduleReminder(actor, msg["payload"], msg["content-type"], msg["accepts"])
	if err != nil {
		reply = Reply{StatusCode: http.StatusBadRequest, Payload: err.Error(), ContentType: "text/plain"}
	} else {
		reply = Reply{StatusCode: http.StatusOK, Payload: "OK", ContentType: "text/plain"}
	}
	return respond(ctx, msg, reply)
}

func tell(ctx context.Context, msg map[string]string) error {
	res, err := invoke(ctx, "POST", msg)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		logger.Error("failed to post to %s: %v", msg["path"], err)
	} else {
		discard(res.Body)
	}
	return nil
}

func dispatch(ctx context.Context, cancel context.CancelFunc, msg map[string]string) error {
	switch msg["command"] {
	case "call":
		return call(ctx, msg)
	case "callback":
		return callback(ctx, msg)
	case "cancel":
		cancel() // never fails
	case "reminder:cancel":
		return reminderCancel(ctx, msg)
	case "reminder:get":
		return reminderGet(ctx, msg)
	case "reminder:schedule":
		return reminderSchedule(ctx, msg)
	case "tell":
		return tell(ctx, msg)
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
		logger.Error("failed to forward message to actor %v: %v", Actor{Type: msg["type"], ID: msg["id"]}, err)
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
		if strings.HasPrefix(msg["command"], "reminder:") { // TODO temporary hack
			err = dispatch(ctx, cancel, msg)
		} else {
			actor := Actor{Type: msg["type"], ID: msg["id"]}
			e, fresh, _ := actor.acquire(ctx)
			if e == nil && ctx.Err() == nil {
				err = forwardToActor(ctx, msg)
			} else {
				defer e.release()
				if fresh {
					activate(ctx, actor)
				}
				msg["path"] = "/actor/" + actor.Type + "/" + actor.ID + msg["path"]
				err = dispatch(ctx, cancel, msg)
			}
		}
	}
	if err == nil {
		message.Mark()
	}
}

// actors

// activate an actor
func activate(ctx context.Context, actor Actor) {
	logger.Debug("activating actor %v", actor)
	res, err := invoke(ctx, "GET", map[string]string{"path": "/actor/" + actor.Type + "/" + actor.ID})
	if err != nil {
		logger.Error("failed to activate actor %v: %v", actor, err)
	} else if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNotFound {
		logger.Error("failed to activate actor %v: %s", actor, ReadAll(res.Body))
	} else {
		discard(res.Body)
	}
}

// deactivate an actor (but retains placement)
func deactivate(ctx context.Context, actor Actor) {
	logger.Debug("deactivating actor %v", actor)
	res, err := invoke(ctx, "DELETE", map[string]string{"path": "/actor/" + actor.Type + "/" + actor.ID})
	if err != nil {
		logger.Error("failed to deactivate actor %v: %v", actor, err)
	} else if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNotFound {
		logger.Error("failed to deactivate actor %v: %s", actor, ReadAll(res.Body))
	} else {
		discard(res.Body)
	}
}

// Collect periodically collect actors with no recent usage (but retains placement)
func Collect(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case now := <-ticker.C:
			logger.Debug("starting collection")
			collect(ctx, now.Add(-10*time.Second))
			logger.Debug("finishing collection")
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}

// ProcessReminders runs periodically and schedules delivery of all reminders whose deadline has passed
func ProcessReminders(ctx context.Context) {
	ticker := time.NewTicker(config.ActorReminderInterval)
	for {
		select {
		case now := <-ticker.C:
			processReminders(ctx, now)
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}
