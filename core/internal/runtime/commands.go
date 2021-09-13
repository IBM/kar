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

	"github.com/IBM/kar/core/internal/config"
	"github.com/IBM/kar/core/internal/pubsub"
	"github.com/IBM/kar/core/pkg/logger"
	"github.com/IBM/kar/core/pkg/redis"
	"github.com/google/uuid"
)

const (
	actorRuntimeRoutePrefix = "/kar/impl/v1/actor/"
)

var (
	// pending requests: map request uuid (string) to channel (chan Reply)
	requests = sync.Map{}
)

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
func TellActor(ctx context.Context, actor Actor, path, payload string, direct bool) error {
	return pubsub.Send(ctx, direct, map[string]string{
		"protocol": "actor",
		"type":     actor.Type,
		"id":       actor.ID,
		"command":  "tell", // post with no callback expected
		"path":     path,
		"payload":  payload})
}

// DeleteActor sends a delete message to an actor and does not wait for a reply
func DeleteActor(ctx context.Context, actor Actor, direct bool) error {
	return pubsub.Send(ctx, direct, map[string]string{
		"protocol": "actor",
		"type":     actor.Type,
		"id":       actor.ID,
		"command":  "delete"})
}

func tellBinding(ctx context.Context, kind string, actor Actor, partition, bindingID string) error {
	return pubsub.Send(ctx, false, map[string]string{
		"protocol":  "actor",
		"type":      actor.Type,
		"id":        actor.ID,
		"command":   "binding:tell",
		"kind":      kind,
		"partition": partition,
		"bindingId": bindingID})
}

// Reply represents the return value of a call
type Reply struct {
	StatusCode  int
	ContentType string
	Payload     string
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

// CallService calls a service and waits for a reply
func CallService(ctx context.Context, service, path, payload, header, method string, direct bool) (*Reply, error) {
	msg := map[string]string{
		"protocol": "service",
		"service":  service,
		"command":  "call",
		"path":     path,
		"header":   header,
		"method":   method,
		"payload":  payload}
	return callHelper(ctx, msg, direct)
}

// CallPromiseService calls a service and returns a request id
func CallPromiseService(ctx context.Context, service, path, payload, header, method string, direct bool) (string, error) {
	msg := map[string]string{
		"protocol": "service",
		"service":  service,
		"command":  "call",
		"path":     path,
		"header":   header,
		"method":   method,
		"payload":  payload}
	return callPromiseHelper(ctx, msg, direct)
}

// CallActor calls an actor and waits for a reply
func CallActor(ctx context.Context, actor Actor, path, payload, session string, direct bool) (*Reply, error) {
	msg := map[string]string{
		"protocol": "actor",
		"type":     actor.Type,
		"id":       actor.ID,
		"command":  "call",
		"path":     path,
		"session":  session,
		"payload":  payload}
	return callHelper(ctx, msg, direct)
}

// CallPromiseActor calls an actor and returns a request id
func CallPromiseActor(ctx context.Context, actor Actor, path, payload string, direct bool) (string, error) {
	msg := map[string]string{
		"protocol": "actor",
		"type":     actor.Type,
		"id":       actor.ID,
		"command":  "call",
		"path":     path,
		"payload":  payload}
	return callPromiseHelper(ctx, msg, direct)
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
	return callHelper(ctx, msg, false)
}

// helper methods to handle incoming messages
// log ignored errors to logger.Error

func respond(ctx context.Context, msg map[string]string, reply *Reply) error {
	err := pubsub.Send(ctx, msg["direct"] == "true", map[string]string{
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
	if !pubsub.IsLiveSidecar(msg["from"]) {
		logger.Info("Cancelling %s from dead sidecar %s", msg["method"], msg["from"])
		return nil
	}

	reply, err := invoke(ctx, msg["method"], msg, msg["metricLabel"])
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
	found := deleteBindings(ctx, msg["kind"], actor, msg["bindingId"])
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
	code, err := putBinding(ctx, msg["kind"], actor, msg["bindingId"], msg["payload"])
	if err != nil {
		reply = &Reply{StatusCode: code, Payload: err.Error(), ContentType: "text/plain"}
	} else {
		reply = &Reply{StatusCode: code, Payload: "OK", ContentType: "text/plain"}
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
	reply, err := invoke(ctx, msg["method"], msg, msg["metricLabel"])
	if err != nil {
		if err != ctx.Err() {
			logger.Debug("tell failed to invoke %s: %v", msg["path"], err)
		}
		return err
	}

	// Examine the reply and log any that represent appliction-level errors.
	// We do this because a tell does not have a caller to which such reporting can be delegated.
	if reply.StatusCode == http.StatusNoContent {
		logger.Debug("Asynchronous invoke of %s returned void", msg["path"])
	} else if reply.StatusCode == http.StatusOK {
		if strings.HasPrefix(reply.ContentType, "application/kar+json") {
			var result actorCallResult
			if err := json.Unmarshal([]byte(reply.Payload), &result); err != nil {
				logger.Error("Asynchronous invoke of %s had malformed result. %v", msg["path"], err)
			} else {
				if result.Error {
					logger.Error("Asynchronous invoke of %s raised error %s", msg["path"], result.Message)
					logger.Error("Stacktrace: %v", result.Stack)
				} else {
					logger.Debug("Asynchronous invoke of %s returned %v", msg["path"], result.Value)
				}
			}
		} else {
			logger.Error("Asynchronous invoke of %s returned unexpected Content-Type %v", msg["path"], reply.ContentType)
		}
	} else {
		logger.Error("Asynchronous invoke of %s returned status %v with body %s", msg["path"], reply.StatusCode, reply.Payload)
	}

	return nil
}

// Returns information about this sidecar's actors
func getActorInformation(ctx context.Context, msg map[string]string) error {
	actorInfo := getMyActiveActors(msg["actorType"])
	m, err := json.Marshal(actorInfo)
	var reply *Reply
	if err != nil {
		logger.Debug("Error marshaling actor information data: %v", err)
		reply = &Reply{StatusCode: http.StatusInternalServerError}
	} else {
		reply = &Reply{StatusCode: http.StatusOK, Payload: string(m), ContentType: "application/json"}
	}
	return respond(ctx, msg, reply)
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
	case "getActiveActors":
		return getActorInformation(ctx, msg)
	default:
		logger.Error("unexpected command %s", msg["command"]) // dropping message
	}
	return nil
}

func forwardToSidecar(ctx context.Context, msg map[string]string) error {
	err := pubsub.Send(ctx, false, msg)
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
			msg["metricLabel"] = msg["service"] + ":" + msg["path"]
			err = dispatch(ctx, cancel, msg)
		} else {
			err = pubsub.Send(ctx, false, msg)
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
			} else if msg["command"] == "delete" {
				session = "exclusive"
			} else {
				session = uuid.New().String() // start new session
			}
		}
		var e *actorEntry
		var fresh bool
		e, fresh, err = actor.acquire(ctx, session)
		if err == errActorHasMoved {
			err = pubsub.Send(ctx, false, msg) // forward
		} else if err == errActorAcquireTimeout {
			payload := fmt.Sprintf("acquiring actor %v timed out, aborting command %s with path %s in session %s", actor, msg["command"], msg["path"], session)
			logger.Error("%s", payload)
			if msg["command"] == "call" {
				var reply *Reply = &Reply{StatusCode: http.StatusRequestTimeout, Payload: payload, ContentType: "text/plain"}
				err = respond(ctx, msg, reply)
			} else {
				err = nil
			}
		} else if err == nil {
			if session == "reminder" { // do not activate actor
				err = dispatch(ctx, cancel, msg)
				e.release(session, false)
				break
			}

			if msg["command"] == "delete" {
				// delete SDK-level in-memory state
				if !fresh {
					deactivate(ctx, actor)
				}
				// delete persistent actor state
				if _, err := redis.Del(ctx, stateKey(actor.Type, actor.ID)); err != nil && err != redis.ErrNil {
					logger.Error("deleting persistent state of %v failed with %v", actor, err)
				}
				// clear placement data and sidecar's in-memory state
				err = e.migrate("")
				if err != nil {
					logger.Error("deleting placement date for %v failed with %v", actor, err)
				}
				break
			}

			var reply *Reply
			if fresh {
				reply, err = activate(ctx, actor)
			}
			if reply != nil { // activate returned an error, report or log error, do not retry
				if msg["command"] == "call" {
					logger.Debug("activate %v returned status %v with body %s, aborting call %s", actor, reply.StatusCode, reply.Payload, msg["path"])
					err = respond(ctx, msg, reply) // return activation error to caller
				} else {
					logger.Error("activate %v returned status %v with body %s, aborting tell %s", actor, reply.StatusCode, reply.Payload, msg["path"])
					err = nil // not to be retried
				}
				e.release(session, false)
			} else if err != nil { // failed to invoke activate
				e.release(session, false)
			} else { // invoke actor method
				msg["metricLabel"] = actor.Type + ":" + msg["path"]
				msg["path"] = actorRuntimeRoutePrefix + actor.Type + "/" + actor.ID + "/" + session + msg["path"]
				msg["content-type"] = "application/kar+json"
				msg["method"] = "POST"
				err = dispatch(ctx, cancel, msg)
				e.release(session, true)
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
	reply, err := invoke(ctx, "GET", map[string]string{"path": actorRuntimeRoutePrefix + actor.Type + "/" + actor.ID}, actor.Type+":activate")
	if err != nil {
		if err != ctx.Err() {
			logger.Debug("activate failed to invoke %s: %v", actorRuntimeRoutePrefix+actor.Type+"/"+actor.ID, err)
		}
		return nil, err
	}
	if reply.StatusCode >= http.StatusBadRequest {
		return reply, nil
	}
	logger.Debug("activate %v returned status %v with body %s", actor, reply.StatusCode, reply.Payload)
	return nil, nil
}

// deactivate an actor (but retains placement)
func deactivate(ctx context.Context, actor Actor) error {
	reply, err := invoke(ctx, "DELETE", map[string]string{"path": actorRuntimeRoutePrefix + actor.Type + "/" + actor.ID}, actor.Type+":deactivate")
	if err != nil {
		if err != ctx.Err() {
			logger.Debug("deactivate failed to invoke %s: %v", actorRuntimeRoutePrefix+actor.Type+"/"+actor.ID, err)
		}
		return err
	}
	if reply.StatusCode >= http.StatusBadRequest {
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
				collect(ctx, now.Add(-config.ActorCollectorInterval))
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

// ValidateActorConfig checks to make sure the user process actually supports
// all the Actor types that were specified with `-actors` when the sidecar was launched.
func ValidateActorConfig(ctx context.Context) {
	for _, actorType := range config.ActorTypes {
		reply, err := invoke(ctx, "HEAD", map[string]string{"path": actorRuntimeRoutePrefix + actorType}, "")
		if err != nil {
			if err != ctx.Err() {
				logger.Error("validate actor type failed for %s: %v", actorType, err)
			}
		} else if reply.StatusCode != http.StatusOK {
			logger.Error("Actor type %v is not recognized by application process!", actorType)
		}
	}
}

// Migrate migrates an actor and associated reminders to a new sidecar
// NOTE: This method is currently unused and not exposed via the KAR REST API.
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
