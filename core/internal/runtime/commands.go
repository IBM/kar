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
	"github.com/IBM/kar/core/internal/rpc"
	"github.com/IBM/kar/core/pkg/logger"
	"github.com/IBM/kar/core/pkg/store"
	"github.com/google/uuid"
)

var (
	// pending requests: map request uuid (string) to channel (chan rpc.Result)
	requests = sync.Map{}
)

const (
	actorRuntimeRoutePrefix = "/kar/impl/v1/actor/"

	actorEndpoint   = "handlerActor"
	serviceEndpoint = "handlerService"
	sidecarEndpoint = "handlerSidecar"
)

func init() {
	rpc.Register(actorEndpoint, handlerActor)
	rpc.Register(serviceEndpoint, handlerService)
	rpc.Register(sidecarEndpoint, handlerSidecar)
}

// Reply represents the return value of a call  TODO: Move back to runtime (commands.go)
type Reply struct {
	StatusCode  int
	ContentType string
	Payload     string
}

////////////////////
// Caller (sending) side of RPCs
////////////////////

func defaultTimeout() time.Time {
	if config.MissingComponentTimeout > 0 {
		return time.Now().Add(config.MissingComponentTimeout)
	} else {
		return time.Time{}
	}
}

// CallService calls a service and waits for a reply
func CallService(ctx context.Context, service, path, payload, header, method string) (*Reply, error) {
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
		bytes, err = rpc.Call(ctx, rpc.Service{Name: service}, serviceEndpoint, defaultTimeout(), bytes)
		if err != nil {
			return nil, err
		}
		var reply Reply
		err = json.Unmarshal(bytes, &reply)
		return &reply, err
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
	}
	requestID, ch, err := rpc.Async(ctx, rpc.Service{Name: service}, serviceEndpoint, defaultTimeout(), bytes)
	if err == nil {
		requests.Store(requestID, ch)
	}
	return requestID, err
}

// CallActor calls an actor and waits for a reply
func CallActor(ctx context.Context, actor Actor, path, payload, session string) (*Reply, error) {
	msg := map[string]string{
		"command": "call",
		"path":    path,
		"session": session,
		"payload": payload}
	bytes, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	} else {
		bytes, err = rpc.Call(ctx, rpc.Session{Name: actor.Type, ID: actor.ID}, actorEndpoint, defaultTimeout(), bytes)
		if err != nil {
			return nil, err
		}
		var reply Reply
		err = json.Unmarshal(bytes, &reply)
		return &reply, err
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
	}

	requestID, ch, err := rpc.Async(ctx, rpc.Session{Name: actor.Type, ID: actor.ID}, actorEndpoint, defaultTimeout(), bytes)
	if err == nil {
		requests.Store(requestID, ch)
	}
	return requestID, err
}

// AwaitPromise awaits the response to an actor or service call
func AwaitPromise(ctx context.Context, requestID string) ([]byte, error) {
	if ch, ok := requests.Load(requestID); ok {
		defer requests.Delete(requestID)
		defer rpc.Reclaim(requestID)
		select {
		case r := <-ch.(<-chan rpc.Result):
			return r.Value, r.Err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, fmt.Errorf("unexpected request %s", requestID)
}

// Bindings sends a binding command (cancel, get, schedule) to an actor's assigned sidecar and waits for a reply
func Bindings(ctx context.Context, kind string, actor Actor, bindingID, nilOnAbsent, action, payload, contentType, accept string) (*Reply, error) {
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
		bytes, err = rpc.Call(ctx, rpc.Session{Name: actor.Type, ID: actor.ID}, actorEndpoint, defaultTimeout(), bytes)
		if err != nil {
			return nil, err
		}
		var reply Reply
		err = json.Unmarshal(bytes, &reply)
		return &reply, err
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
		return rpc.Tell(ctx, rpc.Service{Name: service}, serviceEndpoint, defaultTimeout(), bytes)
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
		return rpc.Tell(ctx, rpc.Session{Name: actor.Type, ID: actor.ID}, actorEndpoint, defaultTimeout(), bytes)
	}
}

// DeleteActor sends a delete message to an actor and does not wait for a reply
func DeleteActor(ctx context.Context, actor Actor) error {
	msg := map[string]string{"command": "delete"}
	bytes, err := json.Marshal(msg)
	if err != nil {
		return err
	} else {
		return rpc.Tell(ctx, rpc.Session{Name: actor.Type, ID: actor.ID}, actorEndpoint, defaultTimeout(), bytes)
	}
}

// LoadBinding sends a binding:load message to the target actor
func LoadBinding(ctx context.Context, kind string, actor Actor, partition int32, bindingID string) error {
	msg := map[string]string{
		"command":   "binding:load",
		"kind":      kind,
		"partition": strconv.Itoa(int(partition)),
		"bindingId": bindingID}
	bytes, err := json.Marshal(msg)
	if err != nil {
		return err
	} else {
		return rpc.Tell(ctx, rpc.Session{Name: actor.Type, ID: actor.ID}, actorEndpoint, time.Time{}, bytes)
	}
}

////////////////////
// Callee (receiving) side of RPCs
////////////////////

func call(ctx context.Context, msg map[string]string) ([]byte, error) {
	reply, err := invoke(ctx, msg["method"], msg, msg["metricLabel"])
	if err != nil {
		if err != ctx.Err() {
			logger.Debug("call failed to invoke %s: %v", msg["path"], err)
		}
		return nil, err
	}

	if reply == nil {
		return nil, nil
	} else {
		return json.Marshal(*reply)
	}
}

func bindingDel(ctx context.Context, actor Actor, msg map[string]string) ([]byte, error) {
	var reply Reply
	found := deleteBindings(ctx, msg["kind"], actor, msg["bindingId"])
	if found == 0 && msg["bindingId"] != "" && msg["nilOnAbsent"] != "true" {
		reply = Reply{StatusCode: http.StatusNotFound}
	} else {
		reply = Reply{StatusCode: http.StatusOK, Payload: strconv.Itoa(found), ContentType: "text/plain"}
	}
	return json.Marshal(reply)
}

func bindingGet(ctx context.Context, actor Actor, msg map[string]string) ([]byte, error) {
	var reply Reply
	found := getBindings(msg["kind"], actor, msg["bindingId"])
	var responseBody interface{} = found
	if msg["bindingId"] != "" {
		if len(found) == 0 {
			if msg["nilOnAbsent"] != "true" {
				reply = Reply{StatusCode: http.StatusNotFound}
				return json.Marshal(reply)
			}
			responseBody = nil
		} else {
			responseBody = found[0]
		}
	}
	blob, err := json.Marshal(responseBody)
	if err != nil {
		reply = Reply{StatusCode: http.StatusInternalServerError, Payload: err.Error(), ContentType: "text/plain"}
	} else {
		reply = Reply{StatusCode: http.StatusOK, Payload: string(blob), ContentType: "application/json"}
	}
	return json.Marshal(reply)
}

func bindingSet(ctx context.Context, actor Actor, msg map[string]string) ([]byte, error) {
	var reply Reply
	code, err := putBinding(ctx, msg["kind"], actor, msg["bindingId"], msg["payload"])
	if err != nil {
		reply = Reply{StatusCode: code, Payload: err.Error(), ContentType: "text/plain"}
	} else {
		reply = Reply{StatusCode: code, Payload: "OK", ContentType: "text/plain"}
	}
	return json.Marshal(reply)
}

func bindingLoad(ctx context.Context, actor Actor, msg map[string]string) error {
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
func getActorInformation(ctx context.Context, msg map[string]string) ([]byte, error) {
	actorInfo := getMyActiveActors(msg["actorType"])
	m, err := json.Marshal(actorInfo)
	var reply Reply
	if err != nil {
		logger.Debug("Error marshaling actor information data: %v", err)
		reply = Reply{StatusCode: http.StatusInternalServerError}
	} else {
		reply = Reply{StatusCode: http.StatusOK, Payload: string(m), ContentType: "application/json"}
	}
	return json.Marshal(reply)
}

func handlerService(ctx context.Context, target rpc.Target, value []byte) ([]byte, error) {
	targetAsService, ok := target.(rpc.Service)
	if !ok {
		return nil, fmt.Errorf("Protocol mismatch: handlerService with target %v", target)
	}
	var msg map[string]string
	err := json.Unmarshal(value, &msg)
	if err != nil {
		return nil, err
	}

	msg["metricLabel"] = targetAsService.Name + ":" + msg["path"]

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

func handlerSidecar(ctx context.Context, target rpc.Target, value []byte) ([]byte, error) {
	_, ok := target.(rpc.Node)
	if !ok {
		return nil, fmt.Errorf("Protocol mismatch: handlerSidecar with target %v", target)
	}
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

func handlerActor(ctx context.Context, target rpc.Target, value []byte) ([]byte, error) {
	targetAsSession, ok := target.(rpc.Session)
	if !ok {
		return nil, fmt.Errorf("Protocol mismatch: handlerSidecar with target %v", target)
	}
	actor := Actor{Type: targetAsSession.Name, ID: targetAsSession.ID}
	var reply []byte = nil
	var err error = nil
	var msg map[string]string

	err = json.Unmarshal(value, &msg)
	if err != nil {
		return nil, err
	}

	// Determine session to use when acquiring actor instance lock
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
	var reason map[string]string
	e, fresh, err, reason = actor.acquire(ctx, session, msg)
	if err != nil {
		if err == errActorHasMoved {
			// TODO: This code path will not possible with the new rpc library; eventually delete this branch
			err = rpc.Tell(ctx, target, actorEndpoint, time.Time{}, value) // forward
			return nil, nil
		} else if err == errActorAcquireTimeout {
			payload := fmt.Sprintf("acquiring actor %v timed out, aborting command %s with path %s in session %s, due to %v", actor, msg["command"], msg["path"], session, reason)
			logger.Error("%s", payload)
			return json.Marshal(Reply{StatusCode: http.StatusRequestTimeout, Payload: payload, ContentType: "text/plain"})
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
		case "binding:load":
			reply = nil
			err = bindingLoad(ctx, actor, msg)
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
		err = e.delete()
		if err != nil {
			logger.Error("deleting placement date for %v failed with %v", actor, err)
		}
		return reply, err
	}

	if fresh {
		reply, err = activate(ctx, actor, msg["command"] == "call", msg["path"])
	}
	if reply != nil { // activate returned an application-level error, do not retry
		err = nil // Disable retry
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
func activate(ctx context.Context, actor Actor, isCall bool, causingMethod string) ([]byte, error) {
	reply, err := invoke(ctx, "GET", map[string]string{"path": actorRuntimeRoutePrefix + actor.Type + "/" + actor.ID}, actor.Type+":activate")
	if err != nil {
		if err != ctx.Err() {
			logger.Debug("activate failed to invoke %s: %v", actorRuntimeRoutePrefix+actor.Type+"/"+actor.ID, err)
		}
		return nil, err
	}
	if reply.StatusCode >= http.StatusBadRequest {
		if isCall {
			logger.Debug("activate %v returned status %v with body %s, aborting call %s", actor, reply.StatusCode, reply.Payload, causingMethod)
		} else {
			// Log at error level becasue there is no one waiting on the method reponse to notice the failure.
			logger.Error("activate %v returned status %v with body %s, aborting tell %s", actor, reply.StatusCode, reply.Payload, causingMethod)
		}
		return json.Marshal(reply)
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
		partitions, rebalance := rpc.GetPartitions()
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
