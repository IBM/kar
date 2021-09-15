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
	"time"

	"github.com/IBM/kar/core/internal/config"
	"github.com/IBM/kar/core/internal/pubsub"
	"github.com/IBM/kar/core/internal/rpc"
	"github.com/IBM/kar/core/pkg/logger"
	"github.com/IBM/kar/core/pkg/store"
	"github.com/google/uuid"
)

const (
	actorRuntimeRoutePrefix = "/kar/impl/v1/actor/"
)

// CallService calls a service and waits for a reply
func CallService(ctx context.Context, service, path, payload, header, method string) (*rpc.Reply, error) {
	msg := map[string]string{
		"command": "call",
		"path":    path,
		"header":  header,
		"method":  method,
		"payload": payload}
	return rpc.CallSidecar(ctx,
		pubsub.KarMsgTarget{Protocol: "service", Name: service},
		pubsub.KarMsgBody{Msg: msg})
}

// CallPromiseService calls a service and returns a request id
func CallPromiseService(ctx context.Context, service, path, payload, header, method string) (string, error) {
	msg := map[string]string{
		"command": "call",
		"path":    path,
		"header":  header,
		"method":  method,
		"payload": payload}
	return rpc.CallPromiseSidecar(ctx,
		pubsub.KarMsgTarget{Protocol: "service", Name: service},
		pubsub.KarMsgBody{Msg: msg})
}

// CallActor calls an actor and waits for a reply
func CallActor(ctx context.Context, actor Actor, path, payload, session string) (*rpc.Reply, error) {
	msg := map[string]string{
		"command": "call",
		"path":    path,
		"session": session,
		"payload": payload}
	return rpc.CallSidecar(ctx,
		pubsub.KarMsgTarget{Protocol: "actor", Name: actor.Type, ID: actor.ID},
		pubsub.KarMsgBody{Msg: msg})
}

// CallPromiseActor calls an actor and returns a request id
func CallPromiseActor(ctx context.Context, actor Actor, path, payload string) (string, error) {
	msg := map[string]string{
		"command": "call",
		"path":    path,
		"payload": payload}
	return rpc.CallPromiseSidecar(ctx,
		pubsub.KarMsgTarget{Protocol: "actor", Name: actor.Type, ID: actor.ID},
		pubsub.KarMsgBody{Msg: msg})
}

// Bindings sends a binding command (cancel, get, schedule) to an actor's assigned sidecar and waits for a reply
func Bindings(ctx context.Context, kind string, actor Actor, bindingID, nilOnAbsent, action, payload, contentType, accept string) (*rpc.Reply, error) {
	msg := map[string]string{
		"bindingId":    bindingID,
		"kind":         kind,
		"command":      "binding:" + action,
		"nilOnAbsent":  nilOnAbsent,
		"content-type": contentType,
		"accept":       accept,
		"payload":      payload}
	return rpc.CallSidecar(ctx,
		pubsub.KarMsgTarget{Protocol: "actor", Name: actor.Type, ID: actor.ID},
		pubsub.KarMsgBody{Msg: msg})
}

// helper methods to handle incoming messages
// log ignored errors to logger.Error

func respond(ctx context.Context, msg map[string]string, reply *rpc.Reply) error {
	response := map[string]string{
		"command":      "callback",
		"request":      msg["request"],
		"statusCode":   strconv.Itoa(reply.StatusCode),
		"content-type": reply.ContentType,
		"payload":      reply.Payload}

	err := pubsub.Send(ctx,
		pubsub.KarMsgTarget{Protocol: "sidecar", Node: msg["from"]},
		pubsub.KarMsgBody{Msg: response})

	if err == pubsub.ErrUnknownSidecar {
		logger.Debug("dropping answer to request %s from dead sidecar %s: %v", msg["request"], msg["from"], err)
		return nil
	}
	return err
}

func call(ctx context.Context, target pubsub.KarMsgTarget, msg pubsub.KarMsgBody) error {
	if !pubsub.IsLiveSidecar(msg.Msg["from"]) {
		logger.Info("Cancelling %s from dead sidecar %s", msg.Msg["method"], msg.Msg["from"])
		return nil
	}

	reply, err := invoke(ctx, msg.Msg["method"], msg.Msg, msg.Msg["metricLabel"])
	if err != nil {
		if err != ctx.Err() {
			logger.Debug("call failed to invoke %s: %v", msg.Msg["path"], err)
		}
		return err
	}
	return respond(ctx, msg.Msg, reply)
}

func bindingDel(ctx context.Context, target pubsub.KarMsgTarget, msg pubsub.KarMsgBody) error {
	var reply *rpc.Reply
	actor := Actor{Type: target.Name, ID: target.ID}
	found := deleteBindings(ctx, msg.Msg["kind"], actor, msg.Msg["bindingId"])
	if found == 0 && msg.Msg["bindingId"] != "" && msg.Msg["nilOnAbsent"] != "true" {
		reply = &rpc.Reply{StatusCode: http.StatusNotFound}
	} else {
		reply = &rpc.Reply{StatusCode: http.StatusOK, Payload: strconv.Itoa(found), ContentType: "text/plain"}
	}
	return respond(ctx, msg.Msg, reply)
}

func bindingGet(ctx context.Context, target pubsub.KarMsgTarget, msg pubsub.KarMsgBody) error {
	var reply *rpc.Reply
	actor := Actor{Type: target.Name, ID: target.ID}
	found := getBindings(msg.Msg["kind"], actor, msg.Msg["bindingId"])
	var responseBody interface{} = found
	if msg.Msg["bindingId"] != "" {
		if len(found) == 0 {
			if msg.Msg["nilOnAbsent"] != "true" {
				reply = &rpc.Reply{StatusCode: http.StatusNotFound}
				return respond(ctx, msg.Msg, reply)
			}
			responseBody = nil
		} else {
			responseBody = found[0]
		}
	}
	blob, err := json.Marshal(responseBody)
	if err != nil {
		reply = &rpc.Reply{StatusCode: http.StatusInternalServerError, Payload: err.Error(), ContentType: "text/plain"}
	} else {
		reply = &rpc.Reply{StatusCode: http.StatusOK, Payload: string(blob), ContentType: "application/json"}
	}
	return respond(ctx, msg.Msg, reply)
}

func bindingSet(ctx context.Context, target pubsub.KarMsgTarget, msg pubsub.KarMsgBody) error {
	var reply *rpc.Reply
	actor := Actor{Type: target.Name, ID: target.ID}
	code, err := putBinding(ctx, msg.Msg["kind"], actor, msg.Msg["bindingId"], msg.Msg["payload"])
	if err != nil {
		reply = &rpc.Reply{StatusCode: code, Payload: err.Error(), ContentType: "text/plain"}
	} else {
		reply = &rpc.Reply{StatusCode: code, Payload: "OK", ContentType: "text/plain"}
	}
	return respond(ctx, msg.Msg, reply)
}

func bindingTell(ctx context.Context, target pubsub.KarMsgTarget, msg pubsub.KarMsgBody) error {
	actor := Actor{Type: target.Name, ID: target.ID}
	err := loadBinding(ctx, msg.Msg["kind"], actor, target.Node, msg.Msg["bindingId"])
	if err != nil {
		if err != ctx.Err() {
			logger.Error("load binding failed: %v", err)
		}
	}
	return nil
}

func tell(ctx context.Context, target pubsub.KarMsgTarget, msg pubsub.KarMsgBody) error {
	reply, err := invoke(ctx, msg.Msg["method"], msg.Msg, msg.Msg["metricLabel"])
	if err != nil {
		if err != ctx.Err() {
			logger.Debug("tell failed to invoke %s: %v", msg.Msg["path"], err)
		}
		return err
	}

	// Examine the reply and log any that represent appliction-level errors.
	// We do this because a tell does not have a caller to which such reporting can be delegated.
	if reply.StatusCode == http.StatusNoContent {
		logger.Debug("Asynchronous invoke of %s returned void", msg.Msg["path"])
	} else if reply.StatusCode == http.StatusOK {
		if strings.HasPrefix(reply.ContentType, "application/kar+json") {
			var result actorCallResult
			if err := json.Unmarshal([]byte(reply.Payload), &result); err != nil {
				logger.Error("Asynchronous invoke of %s had malformed result. %v", msg.Msg["path"], err)
			} else {
				if result.Error {
					logger.Error("Asynchronous invoke of %s raised error %s", msg.Msg["path"], result.Message)
					logger.Error("Stacktrace: %v", result.Stack)
				} else {
					logger.Debug("Asynchronous invoke of %s returned %v", msg.Msg["path"], result.Value)
				}
			}
		} else {
			logger.Error("Asynchronous invoke of %s returned unexpected Content-Type %v", msg.Msg["path"], reply.ContentType)
		}
	} else {
		logger.Error("Asynchronous invoke of %s returned status %v with body %s", msg.Msg["path"], reply.StatusCode, reply.Payload)
	}

	return nil
}

// Returns information about this sidecar's actors
func getActorInformation(ctx context.Context, target pubsub.KarMsgTarget, msg pubsub.KarMsgBody) error {
	actorInfo := getMyActiveActors(msg.Msg["actorType"])
	m, err := json.Marshal(actorInfo)
	var reply *rpc.Reply
	if err != nil {
		logger.Debug("Error marshaling actor information data: %v", err)
		reply = &rpc.Reply{StatusCode: http.StatusInternalServerError}
	} else {
		reply = &rpc.Reply{StatusCode: http.StatusOK, Payload: string(m), ContentType: "application/json"}
	}
	return respond(ctx, msg.Msg, reply)
}

func dispatch(ctx context.Context, cancel context.CancelFunc, target pubsub.KarMsgTarget, msg pubsub.KarMsgBody) error {
	switch msg.Msg["command"] {
	case "call":
		return call(ctx, target, msg)
	case "callback":
		return rpc.Callback(ctx, target, msg) // KLUDGE....will fix once the lowest level of handler registration is implemented
	case "cancel":
		cancel() // never fails
	case "binding:del":
		return bindingDel(ctx, target, msg)
	case "binding:get":
		return bindingGet(ctx, target, msg)
	case "binding:set":
		return bindingSet(ctx, target, msg)
	case "binding:tell":
		return bindingTell(ctx, target, msg)
	case "tell":
		return tell(ctx, target, msg)
	case "getActiveActors":
		return getActorInformation(ctx, target, msg)
	default:
		logger.Error("unexpected command %s", msg.Msg["command"]) // dropping message
	}
	return nil
}

func forwardToSidecar(ctx context.Context, target pubsub.KarMsgTarget, msg pubsub.KarMsgBody) error {
	err := pubsub.Send(ctx, target, msg)
	if err == pubsub.ErrUnknownSidecar {
		logger.Debug("dropping message to dead sidecar %s: %v", target.Node, err)
		return nil
	}
	return err
}

// Process processes one incoming message
func Process(ctx context.Context, cancel context.CancelFunc, message pubsub.Message) {
	var msg pubsub.KarMsg
	err := json.Unmarshal(message.Value, &msg)
	if err != nil {
		logger.Error("failed to unmarshal message: %v", err)
		message.Mark()
		return
	}
	switch msg.Target.Protocol {
	case "service":
		if msg.Target.Name == config.ServiceName {
			msg.Body.Msg["metricLabel"] = msg.Target.Name + ":" + msg.Body.Msg["path"]
			err = dispatch(ctx, cancel, msg.Target, msg.Body)
		} else {
			err = pubsub.Send(ctx, msg.Target, msg.Body)
		}
	case "sidecar":
		if msg.Target.Node == config.ID {
			err = dispatch(ctx, cancel, msg.Target, msg.Body)
		} else {
			err = forwardToSidecar(ctx, msg.Target, msg.Body)
		}
	case "partition":
		err = dispatch(ctx, cancel, msg.Target, msg.Body)
	case "actor":
		actor := Actor{Type: msg.Target.Name, ID: msg.Target.ID}
		session := msg.Body.Msg["session"]
		if session == "" {
			if strings.HasPrefix(msg.Body.Msg["command"], "binding:") {
				session = "reminder"
			} else if msg.Body.Msg["command"] == "delete" {
				session = "exclusive"
			} else {
				session = uuid.New().String() // start new session
			}
		}
		var e *actorEntry
		var fresh bool
		e, fresh, err = actor.acquire(ctx, session)
		if err == errActorHasMoved {
			err = pubsub.Send(ctx, msg.Target, msg.Body) // forward
		} else if err == errActorAcquireTimeout {
			payload := fmt.Sprintf("acquiring actor %v timed out, aborting command %s with path %s in session %s", actor, msg.Body.Msg["command"], msg.Body.Msg["path"], session)
			logger.Error("%s", payload)
			if msg.Body.Msg["command"] == "call" {
				var reply *rpc.Reply = &rpc.Reply{StatusCode: http.StatusRequestTimeout, Payload: payload, ContentType: "text/plain"}
				err = respond(ctx, msg.Body.Msg, reply)
			} else {
				err = nil
			}
		} else if err == nil {
			if session == "reminder" { // do not activate actor
				err = dispatch(ctx, cancel, msg.Target, msg.Body)
				e.release(session, false)
				break
			}

			if msg.Body.Msg["command"] == "delete" {
				// delete SDK-level in-memory state
				if !fresh {
					deactivate(ctx, actor)
				}
				// delete persistent actor state
				if _, err := store.Del(ctx, stateKey(actor.Type, actor.ID)); err != nil && err != store.ErrNil {
					logger.Error("deleting persistent state of %v failed with %v", actor, err)
				}
				// clear placement data and sidecar's in-memory state
				err = e.migrate("")
				if err != nil {
					logger.Error("deleting placement date for %v failed with %v", actor, err)
				}
				break
			}

			var reply *rpc.Reply
			if fresh {
				reply, err = activate(ctx, actor)
			}
			if reply != nil { // activate returned an error, report or log error, do not retry
				if msg.Body.Msg["command"] == "call" {
					logger.Debug("activate %v returned status %v with body %s, aborting call %s", actor, reply.StatusCode, reply.Payload, msg.Body.Msg["path"])
					err = respond(ctx, msg.Body.Msg, reply) // return activation error to caller
				} else {
					logger.Error("activate %v returned status %v with body %s, aborting tell %s", actor, reply.StatusCode, reply.Payload, msg.Body.Msg["path"])
					err = nil // not to be retried
				}
				e.release(session, false)
			} else if err != nil { // failed to invoke activate
				e.release(session, false)
			} else { // invoke actor method
				msg.Body.Msg["metricLabel"] = actor.Type + ":" + msg.Body.Msg["path"]
				msg.Body.Msg["path"] = actorRuntimeRoutePrefix + actor.Type + "/" + actor.ID + "/" + session + msg.Body.Msg["path"]
				msg.Body.Msg["content-type"] = "application/kar+json"
				msg.Body.Msg["method"] = "POST"
				err = dispatch(ctx, cancel, msg.Target, msg.Body)
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
func activate(ctx context.Context, actor Actor) (*rpc.Reply, error) {
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
