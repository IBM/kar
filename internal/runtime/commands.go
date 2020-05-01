// Package runtime implements the core sidecar capabilities
package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/internal/pubsub"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

type subscriber struct {
	actor *Actor
	path  string
}

type subscription struct {
	sub    chan subscriber    // channel to update the subscriber
	cancel context.CancelFunc // to cancel subscription
	done   chan struct{}      // to wait for cancellation to complete
}

var (
	// pending requests: map request uuid (string) to channel (chan Reply)
	requests = sync.Map{}

	subscriptions = sync.Map{}
)

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
func CallActor(ctx context.Context, actor Actor, path, payload, contentType, accept, session string) (*Reply, error) {
	msg := map[string]string{
		"protocol":     "actor",
		"type":         actor.Type,
		"id":           actor.ID,
		"command":      "call",
		"path":         path,
		"content-type": contentType,
		"accept":       accept,
		"session":      session,
		"payload":      payload}
	return callHelper(ctx, msg)
}

// Reminders sends a reminder command (cancel, get, schedule) to a reminder's assigned sidecar and waits for a reply
func Reminders(ctx context.Context, actor Actor, action, payload, contentType, accept string) (*Reply, error) {
	target := reminderPartition(actor)
	msg := map[string]string{
		"protocol":     "partition",
		"partition":    strconv.Itoa(int(target)),
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
	if err == pubsub.ErrUnknownSidecar {
		logger.Debug("dropping answer to request %s from dead sidecar %s: %v", msg["request"], msg["from"], err)
		return nil
	}
	return err
}

func call(ctx context.Context, msg map[string]string) error {
	res, err := invoke(ctx, "POST", msg)
	if err != nil {
		if err != ctx.Err() {
			logger.Debug("call failed to invoke %s: %v", msg["path"], err)
		}
		return err
	}
	reply := Reply{StatusCode: res.StatusCode, Payload: ReadAll(res.Body), ContentType: res.Header.Get("Content-Type")}
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
		if err != ctx.Err() {
			logger.Debug("tell failed to invoke %s: %v", msg["path"], err)
		}
		return err
	}
	if res.StatusCode >= http.StatusBadRequest {
		logger.Error("tell returned status %v with body %s", res.StatusCode, ReadAll(res.Body))
	} else {
		logger.Debug("tell returned status %v with body %s", res.StatusCode, ReadAll(res.Body))
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
		logger.Error("unexpected command %s", msg["command"]) // dropping message
	}
	return nil
}

func forwardToSidecar(ctx context.Context, msg map[string]string) error {
	err := pubsub.Send(ctx, msg)
	if err == pubsub.ErrUnknownSidecar {
		logger.Debug("dropping message to dead sidecar %s: %v", msg["sidecar"], err)
		return nil
	}
	return err
}

// Process processes one incoming message
func Process(ctx context.Context, cancel context.CancelFunc, message pubsub.Message) {
	var msg map[string]string
	err := json.Unmarshal(message.Value, &msg)
	if err != nil {
		logger.Error("failed to unmarshal message: %v", err)
		message.Mark()
		return
	}
	switch msg["protocol"] {
	case "service":
		if msg["service"] == config.ServiceName {
			err = dispatch(ctx, cancel, msg)
		} else {
			err = pubsub.Send(ctx, msg)
		}
	case "sidecar":
		if msg["sidecar"] == config.ID {
			err = dispatch(ctx, cancel, msg)
		} else {
			err = forwardToSidecar(ctx, msg)
		}
	case "partition":
		err = dispatch(ctx, cancel, msg)
	case "actor":
		actor := Actor{Type: msg["type"], ID: msg["id"]}
		session := msg["session"]
		if session == "" {
			session = uuid.New().String() // start new session
		}
		var e *actorEntry
		var fresh bool
		e, fresh, err = actor.acquire(ctx, session)
		if err == errActorHasMoved {
			err = pubsub.Send(ctx, msg) // forward
		} else if err == nil {
			defer e.release()
			if fresh {
				err = activate(ctx, actor)
			}
			if err == nil {
				msg["path"] = "/actor/" + actor.Type + "/" + actor.ID + "/" + session + msg["path"]
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
func activate(ctx context.Context, actor Actor) error {
	res, err := invoke(ctx, "GET", map[string]string{"path": "/actor/" + actor.Type + "/" + actor.ID})
	if err != nil {
		if err != ctx.Err() {
			logger.Debug("activate failed to invoke %s: %v", "/actor/"+actor.Type+"/"+actor.ID, err)
		}
		return err
	}
	if res.StatusCode >= http.StatusBadRequest && res.StatusCode != http.StatusNotFound {
		logger.Error("activate %v returned status %v with body %s", actor, res.StatusCode, ReadAll(res.Body))
	} else {
		logger.Debug("activate %v returned status %v with body %s", actor, res.StatusCode, ReadAll(res.Body))
	}
	return nil
}

// deactivate an actor (but retains placement)
func deactivate(ctx context.Context, actor Actor) error {
	res, err := invoke(ctx, "DELETE", map[string]string{"path": "/actor/" + actor.Type + "/" + actor.ID})
	if err != nil {
		if err != ctx.Err() {
			logger.Debug("deactivate failed to invoke %s: %v", "/actor/"+actor.Type+"/"+actor.ID, err)
		}
		return err
	}
	if res.StatusCode >= http.StatusBadRequest && res.StatusCode != http.StatusNotFound {
		logger.Error("deactivate %v returned status %v with body %s", actor, res.StatusCode, ReadAll(res.Body))
	} else {
		logger.Debug("deactivate %v returned status %v with body %s", actor, res.StatusCode, ReadAll(res.Body))
	}
	return nil
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

// ManageReminderPartitions handles updating this sidecar's in-memory reminder data structures after
// rebalancing operations to reflect the new assignment of partitions.
func ManageReminderPartitions(ctx context.Context) {
	var priorPartitions = make([]int32, 0)
	for {
		newPartitions, rebalance := pubsub.Partitions()
		rebalanceReminders(ctx, priorPartitions, newPartitions)
		priorPartitions = newPartitions
		select {
		case <-rebalance:
		case <-ctx.Done():
			return
		}
	}
}

// Unsubscribe unsubscribes from topic
func Unsubscribe(ctx context.Context, topic, options string) (string, error) {
	var m map[string]string
	if options != "" {
		err := json.Unmarshal([]byte(options), &m)
		if err != nil {
			return "", err
		}
	}
	id := m["id"]
	if id == "" {
		id = topic
	}
	if v, ok := subscriptions.Load(id); ok {
		s := v.(subscription)
		s.cancel()
		select {
		case <-ctx.Done():
		case <-s.done:
		}
		subscriptions.Delete(id)
		return "OK", nil
	}
	return "", fmt.Errorf("no subscription with id %s", id)
}

// Subscribe posts each message on a topic to the specified path until cancelled
func Subscribe(ctx context.Context, topic, options string) (string, error) {
	var m map[string]string
	if options != "" {
		err := json.Unmarshal([]byte(options), &m)
		if err != nil {
			return "", err
		}
	}
	id := m["id"]
	if id == "" {
		id = topic
	}
	path := m["path"]
	actorType := m["actorType"]
	actorID := m["actorId"]
	var sub = subscriber{path: path}
	if actorType != "" {
		sub.actor = &Actor{Type: actorType, ID: actorID}
	}
	if v, ok := subscriptions.Load(id); ok {
		s := v.(subscription)
		s.sub <- sub // update subscriber
		return "OK", nil
	}
	c, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	s := subscription{sub: make(chan subscriber, 1), cancel: cancel, done: done}
	subscriptions.Store(id, s)
	ch, err := pubsub.Subscribe(c, topic, id, &pubsub.Options{OffsetOldest: m["oldest"] != ""})
	if err != nil {
		return "", err
	}
	ok := true
	var msg pubsub.Message
	go func() {
		for ok {
			select {
			case msg, ok = <-ch:
				if ok {
					var reply *Reply
					var err error
					if sub.actor != nil {
						reply, err = CallActor(ctx, *sub.actor, sub.path, string(msg.Value), "text/plain", "", "")
					} else {
						var res *http.Response
						res, err = invoke(ctx, "POST", map[string]string{"path": sub.path, "payload": string(msg.Value), "content-type": "text/plain"})
						if res != nil {
							reply = &Reply{StatusCode: res.StatusCode, Payload: ReadAll(res.Body)}
						}
					}
					msg.Mark()
					if err != nil {
						logger.Error("failed to post to %s: %v", sub.path, err)
					} else {
						if reply.StatusCode >= http.StatusBadRequest {
							logger.Error("subscriber returned status %v with body %s", reply.StatusCode, reply.Payload)
						} else {
							logger.Debug("subscriber returned status %v with body %s", reply.StatusCode, reply.Payload)
						}
					}
				}
			case sub = <-s.sub: // updated subscriber
			}
		}
		close(done)
	}()
	return "OK", nil
}
