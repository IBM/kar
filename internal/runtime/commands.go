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

type subscriber struct {
	actor *Actor
	path  string
}

type subscription struct {
	cancel context.CancelFunc // to cancel subscription
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

func tellBinding(ctx context.Context, kind string, actor Actor, partition, bindingId string) error {
	return pubsub.Send(ctx, map[string]string{
		"protocol":  "actor",
		"type":      actor.Type,
		"id":        actor.ID,
		"command":   "binding:tell",
		"kind":      kind,
		"partition": partition,
		"bindingId": bindingId})
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
	ch := make(chan *Reply)
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
		return r, nil
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

// Bindings sends a binding command (cancel, get, schedule) to an actor's assigned sidecar and waits for a reply
func Bindings(ctx context.Context, kind string, actor Actor, bindingID, nilOnAbsent, action, payload, contentType, accept string) (*Reply, error) {
	msg := map[string]string{
		"protocol":     "actor",
		"type":         actor.Type,
		"id":           actor.ID,
		"bindingId":    bindingID,
		"kind":         kind,
		"command":      "binding:" + action,
		"nilOnAbsent":  nilOnAbsent,
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

// helper methods to handle incoming messages
// log ignored errors to logger.Error

func respond(ctx context.Context, msg map[string]string, reply *Reply) error {
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
	reply, err := invoke(ctx, "POST", msg)
	if err != nil {
		if err != ctx.Err() {
			logger.Debug("call failed to invoke %s: %v", msg["path"], err)
		}
		return err
	}
	return respond(ctx, msg, reply)
}

func callback(ctx context.Context, msg map[string]string) error {
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

func bindingDel(ctx context.Context, msg map[string]string) error {
	var reply *Reply
	actor := Actor{Type: msg["type"], ID: msg["id"]}
	found := deleteBindings(msg["kind"], actor, msg["bindingId"])
	if found == 0 && msg["bindingId"] != "" && msg["nilOnAbsent"] != "true" {
		reply = &Reply{StatusCode: http.StatusNotFound}
	} else {
		reply = &Reply{StatusCode: http.StatusOK, Payload: strconv.Itoa(found), ContentType: "text/plain"}
	}
	return respond(ctx, msg, reply)
}

func bindingGet(ctx context.Context, msg map[string]string) error {
	var reply *Reply
	actor := Actor{Type: msg["type"], ID: msg["id"]}
	found := getBindings(msg["kind"], actor, msg["bindingId"])
	var responseBody interface{} = found
	if msg["bindingId"] != "" {
		if len(found) == 0 {
			if msg["nilOnAbsent"] != "true" {
				reply = &Reply{StatusCode: http.StatusNotFound}
				return respond(ctx, msg, reply)
			}
			responseBody = nil
		} else {
			responseBody = found[0]
		}
	}
	blob, err := json.Marshal(responseBody)
	if err != nil {
		reply = &Reply{StatusCode: http.StatusInternalServerError, Payload: err.Error(), ContentType: "text/plain"}
	} else {
		reply = &Reply{StatusCode: http.StatusOK, Payload: string(blob), ContentType: "application/json"}
	}
	return respond(ctx, msg, reply)
}

func bindingSet(ctx context.Context, msg map[string]string) error {
	var reply *Reply
	actor := Actor{Type: msg["type"], ID: msg["id"]}
	err := postBinding(ctx, msg["kind"], actor, msg["bindingId"], msg["payload"])
	if err != nil {
		reply = &Reply{StatusCode: http.StatusBadRequest, Payload: err.Error(), ContentType: "text/plain"}
	} else {
		reply = &Reply{StatusCode: http.StatusOK, Payload: "OK", ContentType: "text/plain"}
	}
	return respond(ctx, msg, reply)
}

func bindingTell(ctx context.Context, msg map[string]string) error {
	actor := Actor{Type: msg["type"], ID: msg["id"]}
	err := loadBinding(ctx, msg["kind"], actor, msg["partition"], msg["bindingId"])
	if err != nil {
		if err != ctx.Err() {
			logger.Error("load binding failed: %v", err)
		}
	}
	return nil
}

func tell(ctx context.Context, msg map[string]string) error {
	reply, err := invoke(ctx, "POST", msg)
	if err != nil {
		if err != ctx.Err() {
			logger.Debug("tell failed to invoke %s: %v", msg["path"], err)
		}
		return err
	}
	if reply.StatusCode >= http.StatusBadRequest {
		logger.Error("tell returned status %v with body %s", reply.StatusCode, reply.Payload)
	} else {
		logger.Debug("tell returned status %v with body %s", reply.StatusCode, reply.Payload)
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
	case "binding:del":
		return bindingDel(ctx, msg)
	case "binding:get":
		return bindingGet(ctx, msg)
	case "binding:set":
		return bindingSet(ctx, msg)
	case "binding:tell":
		return bindingTell(ctx, msg)
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
			if strings.HasPrefix(msg["command"], "binding:") {
				session = "reminder"
			} else {
				session = uuid.New().String() // start new session
			}
		}
		var e *actorEntry
		var fresh bool
		e, fresh, err = actor.acquire(ctx, session)
		if err == errActorHasMoved {
			err = pubsub.Send(ctx, msg) // forward
		} else if err == nil {
			defer e.release(session)
			if session == "reminder" { // do not activate actor
				err = dispatch(ctx, cancel, msg)
				break
			}
			var reply *Reply
			if fresh {
				reply, err = activate(ctx, actor)
			}
			if reply != nil {
				if msg["command"] == "call" {
					logger.Debug("activate %v returned status %v with body %s, aborting call %s", actor, reply.StatusCode, reply.Payload, msg["path"])
					err = respond(ctx, msg, reply) // return activation error to caller
				} else {
					logger.Error("activate %v returned status %v with body %s, aborting tell %s", actor, reply.StatusCode, reply.Payload, msg["path"])
					err = nil // not to be retried
				}
			} else if err == nil {
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
func activate(ctx context.Context, actor Actor) (*Reply, error) {
	reply, err := invoke(ctx, "GET", map[string]string{"path": "/actor/" + actor.Type + "/" + actor.ID})
	if err != nil {
		if err != ctx.Err() {
			logger.Debug("activate failed to invoke %s: %v", "/actor/"+actor.Type+"/"+actor.ID, err)
		}
		return nil, err
	}
	if reply.StatusCode >= http.StatusBadRequest && reply.StatusCode != http.StatusNotFound {
		return reply, nil
	}
	logger.Debug("activate %v returned status %v with body %s", actor, reply.StatusCode, reply.Payload)
	return nil, nil
}

// deactivate an actor (but retains placement)
func deactivate(ctx context.Context, actor Actor) error {
	reply, err := invoke(ctx, "DELETE", map[string]string{"path": "/actor/" + actor.Type + "/" + actor.ID})
	if err != nil {
		if err != ctx.Err() {
			logger.Debug("deactivate failed to invoke %s: %v", "/actor/"+actor.Type+"/"+actor.ID, err)
		}
		return err
	}
	if reply.StatusCode >= http.StatusBadRequest && reply.StatusCode != http.StatusNotFound {
		logger.Error("deactivate %v returned status %v with body %s", actor, reply.StatusCode, reply.Payload)
	} else {
		logger.Debug("deactivate %v returned status %v with body %s", actor, reply.StatusCode, reply.Payload)
	}
	return nil
}

// Collect periodically collect actors with no recent usage (but retains placement)
func Collect(ctx context.Context) {
	lock := make(chan struct{}, 1) // trylock
	ticker := time.NewTicker(config.ActorCollectorInterval)
	for {
		select {
		case now := <-ticker.C:
			select {
			case lock <- struct{}{}:
				logger.Debug("starting collection")
				collect(ctx, now.Add(-config.ActorCollectorInterval))
				logger.Debug("finishing collection")
				<-lock
			default: // skip this collection if collection is already in progress
			}
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}

// ProcessReminders runs periodically and schedules delivery of all reminders whose targetTime has passed
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

// ManageBindings reloads bindings on rebalance
func ManageBindings(ctx context.Context) {
	for {
		partitions, rebalance := pubsub.Partitions()
		if err := loadBindings(ctx, partitions); err != nil {
			// TODO: This should trigger a more orderly shutdown of the sidecar.
			logger.Fatal("Error when loading bindings: %v", err)
		}
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
	contentType := m["contentType"]
	if contentType == "" {
		contentType = "application/cloudevents+json"
	}
	var sub = subscriber{path: path}
	if actorType != "" {
		sub.actor = &Actor{Type: actorType, ID: actorID}
	}
	c, cancel := context.WithCancel(ctx)
	s := subscription{cancel: cancel}
	subscriptions.Store(id, s)
	f := func(msg pubsub.Message) {
		var reply *Reply
		var err error
		if sub.actor != nil {
			reply, err = CallActor(ctx, *sub.actor, sub.path, string(msg.Value), contentType, "", "")
		} else {
			reply, err = invoke(ctx, "POST", map[string]string{"path": sub.path, "payload": string(msg.Value), "content-type": contentType})
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
	_, err := pubsub.Subscribe(c, topic, id, &pubsub.Options{OffsetOldest: m["oldest"] != ""}, f)
	if err != nil {
		return "", err
	}
	return "OK", nil
}

// Migrate migrates an actor and associated reminders to a new sidecar
func Migrate(ctx context.Context, actor Actor, sidecar string) error {
	e, fresh, err := actor.acquire(ctx, "exclusive")
	if err != nil {
		return err
	}
	if !fresh {
		err = deactivate(ctx, actor)
		if err != nil {
			logger.Error("failed to deactivate actor %v before migration: %v", actor, err)
		}
	}
	err = e.migrate(sidecar)
	if err != nil {
		return err
	}
	// migrateReminders(ctx, actor) TODO
	return nil
}
