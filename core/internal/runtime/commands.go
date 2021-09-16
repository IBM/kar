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

	actorEndpoint   = "handlerActor"
	serviceEndpoint = "handlerService"
	sidecarEndpoint = "handlerSidecar"
)

func init() {
	rpc.RegisterKAR(actorEndpoint, handlerActor)
	rpc.RegisterKAR(serviceEndpoint, handlerService)
	rpc.RegisterKAR(sidecarEndpoint, handlerSidecar)
}

////////////////////
// Caller (sending) side of RPCs
////////////////////

// CallService calls a service and waits for a reply
func CallService(ctx context.Context, service, path, payload, header, method string) (*rpc.Reply, error) {
	msg := map[string]string{
		"command": "call",
		"path":    path,
		"header":  header,
		"method":  method,
		"payload": payload}
	bytes, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	} else {
		return rpc.CallKAR(ctx, rpc.KarMsgTarget{Protocol: "service", Name: service}, serviceEndpoint, bytes)
	}
}

// CallPromiseService calls a service and returns a request id
func CallPromiseService(ctx context.Context, service, path, payload, header, method string) (string, error) {
	msg := map[string]string{
		"command": "call",
		"path":    path,
		"header":  header,
		"method":  method,
		"payload": payload}
	bytes, err := json.Marshal(msg)
	if err != nil {
		return "", err
	} else {
		return rpc.CallPromiseKAR(ctx, rpc.KarMsgTarget{Protocol: "service", Name: service}, serviceEndpoint, bytes)
	}
}

// CallActor calls an actor and waits for a reply
func CallActor(ctx context.Context, actor Actor, path, payload, session string) (*rpc.Reply, error) {
	msg := map[string]string{
		"command": "call",
		"path":    path,
		"session": session,
		"payload": payload}
	bytes, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	} else {
		return rpc.CallKAR(ctx, rpc.KarMsgTarget{Protocol: "actor", Name: actor.Type, ID: actor.ID}, actorEndpoint, bytes)
	}
}

// CallPromiseActor calls an actor and returns a request id
func CallPromiseActor(ctx context.Context, actor Actor, path, payload string) (string, error) {
	msg := map[string]string{
		"command": "call",
		"path":    path,
		"payload": payload}
	bytes, err := json.Marshal(msg)
	if err != nil {
		return "", err
	} else {
		return rpc.CallPromiseKAR(ctx, rpc.KarMsgTarget{Protocol: "actor", Name: actor.Type, ID: actor.ID}, actorEndpoint, bytes)
	}
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
	bytes, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	} else {
		return rpc.CallKAR(ctx, rpc.KarMsgTarget{Protocol: "actor", Name: actor.Type, ID: actor.ID}, actorEndpoint, bytes)
	}
}

// TellService sends a message to a service and does not wait for a reply
func TellService(ctx context.Context, service, path, payload, header, method string) error {
	msg := map[string]string{
		"command": "tell", // post with no callback expected
		"path":    path,
		"header":  header,
		"method":  method,
		"payload": payload}
	bytes, err := json.Marshal(msg)
	if err != nil {
		return err
	} else {
		return rpc.TellKAR(ctx, rpc.KarMsgTarget{Protocol: "service", Name: service}, serviceEndpoint, bytes)
	}
}

// TellActor sends a message to an actor and does not wait for a reply
func TellActor(ctx context.Context, actor Actor, path, payload string) error {
	msg := map[string]string{
		"command": "tell", // post with no callback expected
		"path":    path,
		"payload": payload}
	bytes, err := json.Marshal(msg)
	if err != nil {
		return err
	} else {
		return rpc.TellKAR(ctx, rpc.KarMsgTarget{Protocol: "actor", Name: actor.Type, ID: actor.ID}, actorEndpoint, bytes)
	}
}

// DeleteActor sends a delete message to an actor and does not wait for a reply
func DeleteActor(ctx context.Context, actor Actor) error {
	msg := map[string]string{"command": "delete"}
	bytes, err := json.Marshal(msg)
	if err != nil {
		return err
	} else {
		return rpc.TellKAR(ctx, rpc.KarMsgTarget{Protocol: "actor", Name: actor.Type, ID: actor.ID}, actorEndpoint, bytes)
	}
}

func TellBinding(ctx context.Context, kind string, actor Actor, partition int32, bindingID string) error {
	msg := map[string]string{
		"command":   "binding:tell",
		"kind":      kind,
		"bindingId": bindingID}
	bytes, err := json.Marshal(msg)
	if err != nil {
		return err
	} else {
		return rpc.TellKAR(ctx, rpc.KarMsgTarget{Protocol: "actor", Name: actor.Type, ID: actor.ID, Partition: partition}, actorEndpoint, bytes)
	}
}

////////////////////
// Callee (receiving) side of RPCs
////////////////////

func call(ctx context.Context, msg map[string]string) (*rpc.Reply, error) {
	reply, err := invoke(ctx, msg["method"], msg, msg["metricLabel"])
	if err != nil {
		if err != ctx.Err() {
			logger.Debug("call failed to invoke %s: %v", msg["path"], err)
		}
		return nil, err
	}
	return reply, nil
}

func bindingDel(ctx context.Context, actor Actor, msg map[string]string) (*rpc.Reply, error) {
	var reply *rpc.Reply
	found := deleteBindings(ctx, msg["kind"], actor, msg["bindingId"])
	if found == 0 && msg["bindingId"] != "" && msg["nilOnAbsent"] != "true" {
		reply = &rpc.Reply{StatusCode: http.StatusNotFound}
	} else {
		reply = &rpc.Reply{StatusCode: http.StatusOK, Payload: strconv.Itoa(found), ContentType: "text/plain"}
	}
	return reply, nil
}

func bindingGet(ctx context.Context, actor Actor, msg map[string]string) (*rpc.Reply, error) {
	var reply *rpc.Reply
	found := getBindings(msg["kind"], actor, msg["bindingId"])
	var responseBody interface{} = found
	if msg["bindingId"] != "" {
		if len(found) == 0 {
			if msg["nilOnAbsent"] != "true" {
				reply = &rpc.Reply{StatusCode: http.StatusNotFound}
				return reply, nil
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
	return reply, nil
}

func bindingSet(ctx context.Context, actor Actor, msg map[string]string) (*rpc.Reply, error) {
	var reply *rpc.Reply
	code, err := putBinding(ctx, msg["kind"], actor, msg["bindingId"], msg["payload"])
	if err != nil {
		reply = &rpc.Reply{StatusCode: code, Payload: err.Error(), ContentType: "text/plain"}
	} else {
		reply = &rpc.Reply{StatusCode: code, Payload: "OK", ContentType: "text/plain"}
	}
	return reply, nil
}

func bindingTell(ctx context.Context, target rpc.KarMsgTarget, msg map[string]string) error {
	actor := Actor{Type: target.Name, ID: target.ID}
	err := loadBinding(ctx, msg["kind"], actor, target.Node, msg["bindingId"])
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
func getActorInformation(ctx context.Context, msg map[string]string) (*rpc.Reply, error) {
	actorInfo := getMyActiveActors(msg["actorType"])
	m, err := json.Marshal(actorInfo)
	var reply *rpc.Reply
	if err != nil {
		logger.Debug("Error marshaling actor information data: %v", err)
		reply = &rpc.Reply{StatusCode: http.StatusInternalServerError}
	} else {
		reply = &rpc.Reply{StatusCode: http.StatusOK, Payload: string(m), ContentType: "application/json"}
	}
	return reply, nil
}

func handlerService(ctx context.Context, target rpc.KarMsgTarget, value []byte) (*rpc.Reply, error) {
	var msg map[string]string
	err := json.Unmarshal(value, &msg)
	if err != nil {
		return nil, err
	}

	msg["metricLabel"] = target.Name + ":" + msg["path"]

	switch msg["command"] {
	case "call":
		return call(ctx, msg)
	case "tell":
		return nil, tell(ctx, msg)
	default:
		logger.Error("unexpected command %s", msg["command"]) // dropping message
		return nil, nil
	}
}

func handlerSidecar(ctx context.Context, target rpc.KarMsgTarget, value []byte) (*rpc.Reply, error) {
	var msg map[string]string
	err := json.Unmarshal(value, &msg)
	if err != nil {
		return nil, err
	}

	if msg["command"] == "getActiveActors" {
		return getActorInformation(ctx, msg)
	} else {
		logger.Error("unexpected command %s", msg["command"]) // dropping message
		return nil, nil
	}
}

func handlerActor(ctx context.Context, target rpc.KarMsgTarget, value []byte) (*rpc.Reply, error) {
	var reply *rpc.Reply = nil
	var err error = nil
	var msg map[string]string

	err = json.Unmarshal(value, &msg)
	if err != nil {
		return nil, err
	}

	// Determine session to use when acquiring actor instance lock
	actor := Actor{Type: target.Name, ID: target.ID}
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

	// Acquire the actor instance lock
	var e *actorEntry
	var fresh bool
	e, fresh, err = actor.acquire(ctx, session)
	if err != nil {
		if err == errActorHasMoved {
			// TODO: This code path will not possible with the new rpc library; eventually delete this branch
			err = rpc.TellKAR(ctx, target, actorEndpoint, value) // forward
			return nil, nil
		} else if err == errActorAcquireTimeout {
			payload := fmt.Sprintf("acquiring actor %v timed out, aborting command %s with path %s in session %s", actor, msg["command"], msg["path"], session)
			logger.Error("%s", payload)
			reply = &rpc.Reply{StatusCode: http.StatusRequestTimeout, Payload: payload, ContentType: "text/plain"}
			return reply, nil
		} else {
			// An error or cancelation that caused us to fail to acquire the lock.
			return nil, err
		}
	}

	// We now have the lock on the actor instance.
	// All paths must call release before returning, but we can't just defer it becuase we don't know if we did an invoke or not yet

	if session == "reminder" { // do not activate actor
		switch msg["command"] {
		case "binding:del":
			reply, err = bindingDel(ctx, actor, msg)
		case "binding:get":
			reply, err = bindingGet(ctx, actor, msg)
		case "binding:set":
			reply, err = bindingSet(ctx, actor, msg)
		case "binding:tell":
			reply = nil
			err = bindingTell(ctx, target, msg)
		default:
			logger.Error("unexpected command %s", msg["command"]) // dropping message
			reply = nil
			err = nil
		}
		e.release(session, false)
		return reply, err
	}

	if msg["command"] == "delete" {
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
		return reply, err
	}

	if fresh {
		reply, err = activate(ctx, actor)
	}
	if reply != nil { // activate returned an error, report or log error, do not retry
		if msg["command"] == "call" {
			logger.Debug("activate %v returned status %v with body %s, aborting call %s", actor, reply.StatusCode, reply.Payload, msg["path"])
			err = nil
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

		if msg["command"] == "call" {
			reply, err = call(ctx, msg)
		} else if msg["command"] == "tell" {
			reply = nil
			err = tell(ctx, msg)
		} else {
			logger.Error("unexpected command %s", msg["command"]) // dropping message
			reply = nil
			err = nil
		}

		e.release(session, true)
	}

	return reply, err
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
