//
// Copyright IBM Corporation 2020,2022
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
	"sync"
	"time"
	"os"

	"github.com/IBM/kar/core/internal/config"
	"github.com/IBM/kar/core/pkg/logger"
	"github.com/IBM/kar/core/pkg/rpc"
	"github.com/IBM/kar/core/pkg/store"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// TODO: remove this duplicate struct, use Actor instead
type actorTuple_t struct {
	actorId string
	actorType string
}

type breakpointAttrs_t struct {
	actorId string
	actorType string
	path string

	// "request": trigger breakpoint upon method request
	// "response": trigger breakpoint upon method response
	isRequest string

	// flowId -- the id of the flow to break on
	// (used in single-step debugging)
	flowId string
}

type breakpoint_t struct {
	id string
	breakpointType string //"actor", "node", "global"
	deleteOnHit bool
	attrs breakpointAttrs_t
}

type waitInfo_t struct {
	ActorId string `json:"actorId"`
	ActorType string `json:"actorType"`
	BreakpointId string `json:"breakpointId"`
	RequestId string `json:"requestId"`
	FlowId string `json:"flowId"`
	RequestValue string `json:"requestValue"`
	ResponseValue string `json:"responseValue"`
	IsResponse string `json:"isResponse"`
	NodeId string `json:"nodeId"`
}

var (
	// pending requests: map requestId (string) to channel (chan rpc.Result)
	requests = sync.Map{}

	// below: lots of debugger stuff
	// breakpoints map
	breakpoints = map[string]breakpoint_t{}
	breakpointsByAttrs = map[breakpointAttrs_t]breakpoint_t{}
	breakpointsLock = sync.RWMutex{}

	// debugger data structure that allows actors to pause on breakpoints
	isActorPaused = map[actorTuple_t]chan struct{}{}
	pausedBreaks = map[actorTuple_t]breakpoint_t{}
	isActorPausedLock = sync.RWMutex{}
	// is the entire node paused?
	// if so, then every new actor will be individually paused.
	isNodePaused = false
	nodePausedBk breakpoint_t //breakpoint on which whole node paused
	immuneFromNodePaused = map[actorTuple_t]struct{} {}
	isNodePausedLock = sync.RWMutex{}

	// for each of the channels used for waiting, tells us if already
	// closed (this allows us to avoid panics incurred by trying
	// to close closed channels)
	isChannelOpen = map[chan struct{}]bool{}
	isChannelOpenLock = sync.RWMutex{}

	// are any debuggers present?
	isDebuggerPresent = false
	isDebuggerPresentLock = sync.RWMutex {}

	debugConns = map[string]*websocket.Conn {}
	debugConnsLock = sync.Mutex{}

	// list of debugger nodes
	// (this way, when we pause/unpause, we can inform the debugger)
	// why not use a service? because we want to broadcast to all
	// debuggers; services choose randomly a node to serve each req
	debuggersMap = map[string]bool{}
	debuggersMapLock = sync.RWMutex{}

	waitingActors = map[actorTuple_t]waitInfo_t{}
	waitingActorsLock = sync.RWMutex{}

	// an actor is busy if it is in the middle of calling another
	busyInfo = listBusyInfo_t {
		ActorHandling: map[string]actorSentInfo_t{},
		ActorSent: map[string]actorSentInfo_t{},
	}
	busyInfoLock = sync.RWMutex{}
)

const (
	actorRuntimeRoutePrefix = "/kar/impl/v1/actor/"

	actorEndpoint   = "handlerActor"
	bindingEndpoint = "handlerBinding"
	serviceEndpoint = "handlerService"
	sidecarEndpoint = "handlerSidecar"
)

func init() {
	rpc.RegisterSession(actorEndpoint, handlerActor)
	rpc.RegisterSession(bindingEndpoint, handlerBinding)
	rpc.RegisterService(serviceEndpoint, handlerService)
	rpc.RegisterNode(sidecarEndpoint, handlerSidecar)
}

// Reply contains the subset of an http.Response that are relevant to higher levels of the runtime
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

func newFlowId() string {
	return "flow-" + uuid.New().String()
}

func newLockId() string {
	return "dl-" + uuid.New().String()
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
		bytes, err = rpc.Call(ctx, rpc.Destination{Target: rpc.Service{Name: service}, Method: serviceEndpoint}, defaultTimeout(), "", bytes)
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
	requestID, ch, err := rpc.Async(ctx, rpc.Destination{Target: rpc.Service{Name: service}, Method: serviceEndpoint}, defaultTimeout(), bytes)
	if err == nil {
		requests.Store(requestID, ch)
	}
	return requestID, err
}

// CallActor calls an actor and waits for a reply
func CallActor(ctx context.Context, actor Actor, path, payload, flow string, parentID string) (*Reply, error) {
	if flow == "" {
		flow = newFlowId()
	}
	msg := map[string]string{
		"command": "call",
		"path":    path,
		"payload": payload}

	bytes, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	} else {
		//bytes, err = rpc.Call(ctx, rpc.Destination{Target: rpc.Session{Name: actor.Type, ID: actor.ID, Flow: flow}, Method: actorEndpoint}, defaultTimeout(), parentID, bytes)

		// very ugly! reimplement rpc.Call in order to get the request id
		// this is necessary if we want to view indirect pauses in the debugger
		// TODO: potentially fix
		requestID, ch, err := rpc.AsyncParent(ctx, rpc.Destination{Target: rpc.Session{Name: actor.Type, ID: actor.ID, Flow: flow}, Method: actorEndpoint}, defaultTimeout(), parentID, bytes)
		if err != nil {
			return nil, err
		}
		defer requests.Delete(requestID)

		/*accessing isDebuggerPresent without a lock -- risky! but fast*/
		if config.IsDebugMode || isDebuggerPresent {
			busyInfoLock.Lock()
			busyInfo.ActorSent[requestID] = actorSentInfo_t {
				Actor: actor_t {ActorType: actor.Type, ActorId: actor.ID },
				ParentId: parentID,
				FlowId: flow,
				RequestValue: string(bytes),
			}
			busyInfoLock.Unlock()

			deleteSent := func() {
				busyInfoLock.Lock()
				delete(busyInfo.ActorSent, requestID)
				busyInfoLock.Unlock()
			}
			defer deleteSent()
		}

		var bytes []byte
		select {
		case result := <-ch:
			bytes = result.Value
			err = result.Err
		case <-ctx.Done():
			return nil, ctx.Err()
		}

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

	requestID, ch, err := rpc.Async(ctx, rpc.Destination{Target: rpc.Session{Name: actor.Type, ID: actor.ID, Flow: newFlowId()}, Method: actorEndpoint}, defaultTimeout(), bytes)
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
		"command":      action,
		"nilOnAbsent":  nilOnAbsent,
		"content-type": contentType,
		"accept":       accept,
		"payload":      payload}
	bytes, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	} else {
		bytes, err = rpc.Call(ctx, rpc.Destination{Target: rpc.Session{Name: actor.Type, ID: actor.ID, Flow: "nonexclusive"}, Method: bindingEndpoint}, defaultTimeout(), "", bytes)
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
		return rpc.Tell(ctx, rpc.Destination{Target: rpc.Service{Name: service}, Method: serviceEndpoint}, defaultTimeout(), "", bytes)
	}
}

// TellActor sends a message to an actor and does not wait for a reply
func TellActor(ctx context.Context, actor Actor, path, payload string, parentID string) error {
	msg := map[string]string{
		"command": "tell", // post with no callback expected
		"path":    path,
		"payload": payload}
	bytes, err := json.Marshal(msg)
	if err != nil {
		return err
	} else {
		return rpc.Tell(ctx, rpc.Destination{Target: rpc.Session{Name: actor.Type, ID: actor.ID, Flow: newFlowId()}, Method: actorEndpoint}, defaultTimeout(), parentID, bytes)
	}
}

// DeleteActor sends a delete message to an actor and does not wait for a reply
func DeleteActor(ctx context.Context, actor Actor) error {
	msg := map[string]string{"command": "delete"}
	bytes, err := json.Marshal(msg)
	if err != nil {
		return err
	} else {
		return rpc.Tell(ctx, rpc.Destination{Target: rpc.Session{Name: actor.Type, ID: actor.ID, Flow: "deactivate"}, Method: actorEndpoint}, defaultTimeout(), "", bytes)
	}
}

// LoadBinding sends a load message to the bindingEndpoint the sidecar that hosts the target actor
func LoadBinding(ctx context.Context, kind string, actor Actor, partition int32, bindingID string) error {
	msg := map[string]string{
		"command":   "load",
		"kind":      kind,
		"partition": strconv.Itoa(int(partition)),
		"bindingId": bindingID}
	bytes, err := json.Marshal(msg)
	if err != nil {
		return err
	} else {
		return rpc.Tell(ctx, rpc.Destination{Target: rpc.Session{Name: actor.Type, ID: actor.ID, Flow: "nonexclusive"}, Method: bindingEndpoint}, time.Time{}, "", bytes)
	}
}

////////////////////
// Callee (receiving) side of RPCs
////////////////////

func handlerSidecar(ctx context.Context, target rpc.Node, value []byte) ([]byte, error) {
	var msg map[string]string
	err := json.Unmarshal(value, &msg)
	if err != nil {
		return nil, err
	}

	//fmt.Printf("Handler sidecar: %v\n", msg)

	var replyBytes []byte
	var replyErr error
	if msg["command"] == "getActiveActors" {
		replyBytes, replyErr = getActorInformation(ctx, msg)
		return replyBytes, replyErr
	} else if msg["command"] == "getRuntimeAddr" {
		replyBytes, replyErr = getRuntimeAddr(ctx, msg)
		return replyBytes, replyErr
	} else if msg["command"] == "setBreakpoint" {
		//logger.Info("setting breakpoint")
		registerDebugger(msg["srcNodeId"])
		replyBytes, replyErr = setBreakpoint(ctx, msg)
		return replyBytes, replyErr
	} else if msg["command"] == "unsetBreakpoint" {
		registerDebugger(msg["srcNodeId"])
		replyBytes, replyErr = unsetBreakpoint(ctx, msg)
		return replyBytes, replyErr
	} else if msg["command"] == "pause" {
		registerDebugger(msg["srcNodeId"])
		if msg["actorType"] == "" || msg["actorId"] == "" {
			// pause entire node
			replyBytes, replyErr = pauseNode(breakpoint_t{id: msg["breakpointId"]})
		} else {
			info := actorTuple_t {
				actorId: msg["actorId"],
				actorType: msg["actorType"],
			}
			replyBytes, replyErr = pause(info, breakpoint_t{id: msg["breakpointId"]})
		}
		return replyBytes, replyErr
	} else if msg["command"] == "unpause" {
		registerDebugger(msg["srcNodeId"])
		if msg["actorType"] == "" || msg["actorId"] == "" {
			// unpause entire node
			_, onlyUnpauseCurrent := msg["onlyUnpauseCurrent"]
			replyBytes, replyErr = unpauseNode(onlyUnpauseCurrent)
		} else {
			info := actorTuple_t {
				actorId: msg["actorId"],
				actorType: msg["actorType"],
			}
			replyBytes, replyErr = unpause(info)
		}
		return replyBytes, replyErr
	} else if msg["command"] == "registerDebugger" {
		replyBytes, replyErr = registerDebugger(msg["nodeId"])
		return replyBytes, replyErr
	} else if msg["command"] == "unregisterDebugger" {
		replyBytes, replyErr = unregisterDebugger(msg["nodeId"])
		return replyBytes, replyErr
	} else if msg["command"] == "notifyPause" {
		notifyPause(msg) // starts a goroutine
		reply := Reply{StatusCode: http.StatusOK, ContentType: "application/json"}
		return json.Marshal(reply)
	} else if msg["command"] == "notifyBreakpoint" {
		notifyBreakpoint(msg) // starts a goroutine
		reply := Reply{StatusCode: http.StatusOK, ContentType: "application/json"}
		return json.Marshal(reply)
	} else if msg["command"] == "listBreakpoints" {
		replyBytes, replyErr = listBreakpoints()
		return replyBytes, replyErr
	} else if msg["command"] == "listPausedActors" {
		replyBytes, replyErr = listPausedActors()
		return replyBytes, replyErr
	} else if msg["command"] == "listBusyActors" {
		replyBytes, replyErr = listBusyActors()
		return replyBytes, replyErr
	} else {
		logger.Error("unexpected command %s", msg["command"]) // dropping message
		return nil, nil
	}
}

func handlerService(ctx context.Context, target rpc.Service, value []byte) ([]byte, error) {
	var msg map[string]string
	err := json.Unmarshal(value, &msg)
	if err != nil {
		return nil, err
	}

	command := msg["command"]
	if !(command == "call" || command == "tell") {
		logger.Error("unexpected command %s", command)
		return nil, nil // returning `nil` error indicates that message processing is complete (ie, drop unknown commands)
	}

	reply, err := invoke(ctx, msg["method"], msg, target.Name+":"+msg["path"])
	if err != nil {
		if err != ctx.Err() {
			logger.Debug("%s failed to invoke %s: %v", command, msg["path"], err)
		}
		return nil, err
	}

	var replyBytes []byte = nil
	if reply != nil {
		if command == "tell" {
			// reply is dropped after logging non-200 status code; no one is waiting for it.
			if reply.StatusCode >= 300 || reply.StatusCode < 200 {
				logger.Error("Asynchronous %s of %s returned status %v with body %s", msg["method"], msg["path"], reply.StatusCode, reply.Payload)
			}
		} else {
			if msg["actorTailCall"] == "true" {
				// Caller is expecting a result encoded using the kar-actor conventions.
				if reply.StatusCode == 200 {
					var rawVal interface{}
					if e2 := json.Unmarshal([]byte(reply.Payload), &rawVal); e2 == nil {
						wrapped := map[string]interface{}{"value": rawVal}
						tmp, _ := json.Marshal(wrapped)
						reply.Payload = string(tmp)
						reply.ContentType = "application/kar+json"
					} else {
						wrapped := map[string]interface{}{"value": reply.Payload}
						tmp, _ := json.Marshal(wrapped)
						reply.Payload = string(tmp)
						reply.ContentType = "application/kar+json"
					}
				}
			}
			replyBytes, err = json.Marshal(*reply)
		}
	}

	return replyBytes, err
}

func handlerActor(ctx context.Context, target rpc.Session, instance *rpc.SessionInstance, requestID string, value []byte) (*rpc.Destination, []byte, error) {
	actor := Actor{Type: target.Name, ID: target.ID}
	session := target.Flow + ":" + requestID
	var reply []byte = nil
	var debugReply []byte = nil
	var err error = nil
	var msg map[string]string

	err = json.Unmarshal(value, &msg)
	if err != nil {
		return nil, nil, err
	}

	var bkActorId, bkActorType, bkPath string
	/*accessing isDebuggerPresent without a lock -- risky! but fast*/
	if config.IsDebugMode || isDebuggerPresent {
		busyInfoLock.Lock()
		busyInfo.ActorHandling[requestID] = actorSentInfo_t {
			Actor: actor_t {ActorType: actor.Type, ActorId: actor.ID },
			FlowId: target.Flow,
			RequestValue: string(value),
		}
		busyInfoLock.Unlock()

		deleteHandling := func() {
			busyInfoLock.Lock()
			delete(busyInfo.ActorHandling, requestID)
			busyInfoLock.Unlock()
		}
		defer deleteHandling()
	

		bkActorId = target.ID
		bkActorType = target.Name
		bkPath = msg["path"]

		isBreak, bk := checkBreakpoint(breakpointAttrs_t {
			actorId: bkActorId,
			actorType: bkActorType,
			path: bkPath,
			isRequest: "request",
			flowId: target.Flow,
		})

		myActor := actorTuple_t { actorId: target.ID, actorType: target.Name }

		if isBreak {
			//fmt.Printf("Breakpoint %v hit!\n", bk)
			informBreakpoint(myActor, requestID, bk, string(value), "", "request")
			switch bk.breakpointType {
			case "actor":
				pause(actorTuple_t { actorId: target.ID, actorType: target.Name }, bk)
			case "node":
				pauseNode(bk)
			case "global":
				pauseNode(bk)
				pauseAllSidecars(bk)
			case "suicide":
				cancel9()
				for true {}
			}
		}

		isNodePausedLock.Lock()
		_, ok := immuneFromNodePaused[myActor]
		if isNodePaused && !ok {
			pause(myActor, nodePausedBk)
		}
		isNodePausedLock.Unlock()

		// if we're paused, then wait
		waitOnPause(actorTuple_t { actorId: target.ID, actorType: target.Name },
			requestID,
			target.Flow,
			string(value),
			"",
			false,
			!isBreak, //only send a pause if we didn't hit the breakpoint
		)
	}

	if target.Flow != instance.ActiveFlow {
		logger.Error("Flow violation: mismatch between target %v and instance %v at entry", target, instance)
		return nil, nil, fmt.Errorf("Flow violation: mismatch between target %v and instance %v at entry", target, instance)
	}

	if msg["command"] == "delete" {
		// delete SDK-level in-memory state
		if instance.Activated {
			deactivate(ctx, instance)
		}
		// delete persistent actor state
		if _, err := store.Del(ctx, stateKey(actor.Type, actor.ID)); err != nil && err != store.ErrNil {
			logger.Error("deleting persistent state of %v failed with %v", actor, err)
		}
		// clear placement data and sidecar's in-memory state (effectively also releases the lock, since we are deleting the table entry)
		err = rpc.DelSession(ctx, rpc.Session{Name: actor.Type, ID: actor.ID})
		if err != nil {
			logger.Error("deleting placement data for %v failed with %v", actor, err)
		}
		return nil, reply, err
	}

	var dest *rpc.Destination = nil
	if !instance.Activated {
		reply, err = activate(ctx, actor, session, msg)
		if reply != nil {
			// activate returned an application-level error, do not retry
			err = nil
		} else if err == nil {
			instance.Activated = true
		}
	}

	if instance.Activated {
		// invoke actor method
		metricLabel := actor.Type + ":" + msg["path"] // compute metric label before we augment the path with id+flow
		msg["path"] = actorRuntimeRoutePrefix + actor.Type + "/" + actor.ID + msg["path"] + "?session=" + session
		msg["content-type"] = "application/kar+json"
		msg["method"] = "POST"

		command := msg["command"]
		if command == "call" || command == "tell" {
			reply = nil
			replyStruct, err := invoke(ctx, msg["method"], msg, metricLabel)
			if err != nil {
				if err != ctx.Err() {
					logger.Debug("%s failed to invoke %s: %v", command, msg["path"], err)
				}
			} else if replyStruct != nil {
				debugReply, _ = json.Marshal(*replyStruct)
				if command == "tell" {
					// TELL: no waiting caller, so we have to inspect here and figure out if the method returned void, a result, a tail call, or an error
					if replyStruct.StatusCode == http.StatusNoContent {
						// Void return from a tell; nothing further to do.
					} else if replyStruct.StatusCode == http.StatusOK {
						var result actorCallResult
						if err = json.Unmarshal([]byte(replyStruct.Payload), &result); err != nil {
							logger.Error("Asynchronous invoke of %s had malformed result. %v", msg["path"], err)
							err = nil // don't try to rexecute; this is a KAR runtime-level protocol error that should never happen
						} else {
							if result.Error {
								logger.Error("Asynchronous invoke of %s raised error %s\nStacktrace: %v", msg["path"], result.Message, result.Stack)
							} else if result.TailCall {
								cr := result.Value.(map[string]interface{})
								if _, ok := cr["serviceName"]; ok {
									nextService := rpc.Service{Name: cr["serviceName"].(string)}
									dest = &rpc.Destination{Target: nextService, Method: serviceEndpoint}
								} else if _, ok := cr["actorType"]; ok {
									nextActor := rpc.Session{Name: cr["actorType"].(string), ID: cr["actorId"].(string), Flow: target.Flow}
									if nextActor.Name == target.Name && nextActor.ID == target.ID && cr["releaseLock"] != "true" {
										nextActor.DeferredLockID = newLockId()
									}
									dest = &rpc.Destination{Target: nextActor, Method: actorEndpoint}
								} else {
									logger.Error("Asynchronous invoke of %s returned unsupported tail call result %v", msg["path"], cr)
									err = fmt.Errorf("Asynchronous invoke of %s returned unsupported tail call result %v", msg["path"], cr)
								}
								if dest != nil {
									msg := map[string]string{
										"command": "tell",
										"path":    cr["path"].(string),
										"payload": cr["payload"].(string)}
									if cr["method"] != nil {
										msg["method"] = cr["method"].(string)
										msg["header"] = "{\"Content-Type\": [\"application/json\"]}"
										msg["actorTailCall"] = "true"
									}
									reply, err = json.Marshal(msg)
								}
							}
						}
					} else {
						logger.Error("Asynchronous invoke of %s returned status %v with body %s", msg["path"], replyStruct.StatusCode, replyStruct.Payload)
					}
				} else {
					// CALL: there is a waiting caller, so after handling tail calls, anything else (normal or error) is simply passed through.
					if replyStruct.StatusCode == http.StatusOK {
						var result actorCallResult
						if err = json.Unmarshal([]byte(replyStruct.Payload), &result); err == nil && result.TailCall {
							cr := result.Value.(map[string]interface{})
							if _, ok := cr["serviceName"]; ok {
								nextService := rpc.Service{Name: cr["serviceName"].(string)}
								dest = &rpc.Destination{Target: nextService, Method: serviceEndpoint}
							} else if _, ok := cr["actorType"]; ok {
								nextActor := rpc.Session{Name: cr["actorType"].(string), ID: cr["actorId"].(string), Flow: target.Flow}
								if nextActor.Name == target.Name && nextActor.ID == target.ID && cr["releaseLock"] != "true" {
									nextActor.DeferredLockID = newLockId()
								}
								dest = &rpc.Destination{Target: nextActor, Method: actorEndpoint}
							} else {
								err = fmt.Errorf("Invoke of %s returned unsupported tail call result %v", msg["path"], cr)
							}
							if dest != nil {
								msg := map[string]string{
									"command": "call",
									"path":    cr["path"].(string),
									"payload": cr["payload"].(string)}
								if cr["method"] != nil {
									msg["method"] = cr["method"].(string)
									msg["header"] = "{\"Content-Type\": [\"application/json\"]}"
									msg["actorTailCall"] = "true"
								}
								reply, err = json.Marshal(msg)
							}
						}
					}
					if reply == nil {
						// If it wasn't a well-formed continuation then the result of a call is always the replyStruct.
						// We intentionally discard any errors that might have happened while inspecting replyStruct
						// (if there were errors, the caller is better positioned to propagate/report them).
						reply, err = json.Marshal(*replyStruct)
					}
				}
			}
		} else {
			logger.Error("unexpected actor command %s", msg["command"]) // dropping message
			reply = nil
			err = nil
		}
	}

	if target.Flow != instance.ActiveFlow {
		logger.Error("Flow violation: mismatch between target %v and instance %v at exit", target, instance)
	}

	/*accessing isDebuggerPresent without a lock -- risky! but fast*/
	if config.IsDebugMode || isDebuggerPresent {
		// break on response breakpoints
		isBreak, bk := checkBreakpoint(breakpointAttrs_t {
			actorId: bkActorId,
			actorType: bkActorType,
			path: bkPath,
			isRequest: "response",
			flowId: target.Flow,
		})
		
		if reply != nil {
			debugReply = reply
		}

		if isBreak {
			informBreakpoint(actorTuple_t { actorId: target.ID, actorType: target.Name }, requestID, bk, string(value), string(debugReply), "response")
			switch bk.breakpointType {
			case "actor":
				pause(actorTuple_t { actorId: target.ID, actorType: target.Name }, bk)
			case "node":
				pause(actorTuple_t { actorId: "", actorType: ""}, bk)
			case "global":
				pause(actorTuple_t { actorId: "", actorType: ""}, bk)
				pauseAllSidecars(bk)
			case "suicide":
				cancel9()
				for true {}

			}
		}

		// if we're paused, then wait
		waitOnPause(actorTuple_t { actorId: target.ID, actorType: target.Name },
			requestID,
			target.Flow,
			string(value),
			string(debugReply),
			true,
			!isBreak, //only send a pause if we didn't hit the breakpoint.
		)
	}

	return dest, reply, err
}

func handlerBinding(ctx context.Context, target rpc.Session, instance *rpc.SessionInstance, requestID string, value []byte) (*rpc.Destination, []byte, error) {
	actor := Actor{Type: target.Name, ID: target.ID}
	var reply []byte = nil
	var err error = nil
	var msg map[string]string

	err = json.Unmarshal(value, &msg)
	if err != nil {
		return nil, nil, err
	}

	switch msg["command"] {
	case "del":
		reply, err = bindingDel(ctx, actor, msg)
	case "get":
		reply, err = bindingGet(ctx, actor, msg)
	case "set":
		reply, err = bindingSet(ctx, actor, msg)
	case "load":
		reply = nil
		err = bindingLoad(ctx, actor, msg)
	default:
		logger.Error("unexpected binding command %s", msg["command"]) // dropping message
		reply = nil
		err = nil
	}

	return nil, reply, err
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

// Returns information about this sidecar's actors
func getActorInformation(ctx context.Context, msg map[string]string) ([]byte, error) {
	actorInfo := rpc.GetLocalActivatedSessions(ctx, msg["actorType"])
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

// Returns this sidecar's hostname and port
func getRuntimeAddr(ctx context.Context, msg map[string]string) ([]byte, error) {
	replyMap := map[string]interface{} {}
	hostname, err := os.Hostname()
	if err == nil { replyMap["host"] = hostname }
	replyMap["port"] = runtimePortInt

	m, err := json.Marshal(replyMap)

	var reply Reply
	if err != nil {
		logger.Debug("Error marshaling address infromation: %v", err)
		reply = Reply{StatusCode: http.StatusInternalServerError}
	} else {
		reply = Reply{StatusCode: http.StatusOK, Payload: string(m), ContentType: "application/json"}
	}
	return json.Marshal(reply)
}


// sets a breakpoint
func setBreakpoint(ctx context.Context, msg map[string]string) ([]byte, error) {
	flowId, flowOk := msg["flowId"]
	var attrs breakpointAttrs_t
	var fullAttrs = breakpointAttrs_t {
		actorId: msg["actorId"],
		actorType: msg["actorType"],
		path: msg["path"],
		//isCaller: msg["isCaller"],
		isRequest: msg["isRequest"],
	}

	if flowOk && flowId != "" {
		attrs = breakpointAttrs_t {
			flowId: flowId,
		}
	} else {
		attrs = fullAttrs
	}

	breakpoint := breakpoint_t {
		id: msg["breakpointId"],
		breakpointType: msg["breakpointType"],
		attrs: fullAttrs,
	}
	if msg["deleteOnHit"] == "true" {
		breakpoint.deleteOnHit = true
	}

	_, alreadyExists := breakpointsByAttrs[attrs]
	if alreadyExists { goto doReply }

	breakpointsLock.Lock()
	breakpoints[msg["breakpointId"]] = breakpoint
	breakpointsByAttrs[attrs] = breakpoint //msg["breakpointId"]

	//fmt.Printf("Breakpoints after set: %v\n", breakpoints)

	breakpointsLock.Unlock()

doReply:
	reply := Reply{StatusCode: http.StatusOK, ContentType: "application/json"}
	return json.Marshal(reply)
}

func unsetBreakpoint(ctx context.Context, msg map[string]string) ([]byte, error) {
	breakpointsLock.Lock()
	breakpoint := breakpoints[msg["breakpointId"]]
	delete(breakpoints, msg["breakpointId"])
	delete(breakpointsByAttrs, breakpoint.attrs)

	//fmt.Printf("Breakpoints after unset: %v\n", breakpoints)

	breakpointsLock.Unlock()


	reply := Reply{StatusCode: http.StatusOK, ContentType: "application/json"}
	return json.Marshal(reply)
}

func pause(info actorTuple_t, bk breakpoint_t) ([]byte, error) {
	isActorPausedLock.Lock()
	_, ok := isActorPaused[info]
	if !ok {
		isChannelOpenLock.Lock()
		cvar := make(chan struct{})
		isChannelOpen[cvar] = true
		isActorPaused[info] = cvar
		pausedBreaks[info] = bk
		isChannelOpenLock.Unlock()
	}
	//fmt.Printf("paused actors after pause: %v\n", isActorPaused)
	isActorPausedLock.Unlock()

	reply := Reply{StatusCode: http.StatusOK, ContentType: "application/json"}
	return json.Marshal(reply)

}

func pauseNode(bk breakpoint_t) ([]byte, error) {
	isNodePausedLock.Lock()
	nodePausedBk = bk
	for key, _ := range immuneFromNodePaused {
		delete(immuneFromNodePaused, key)
	}

	isNodePaused = true
	nodePausedBk = bk
	isNodePausedLock.Unlock()

	reply := Reply{StatusCode: http.StatusOK, ContentType: "application/json"}
	return json.Marshal(reply)
}

func unpause(info actorTuple_t) ([]byte, error) {
	isActorPausedLock.Lock()
	cvar, ok := isActorPaused[info]
	if !ok {
		// actor not even paused
		goto endUnpause
	}

	isChannelOpenLock.Lock()
	{
		isOpen, openOk := isChannelOpen[cvar]

		if openOk && isOpen {
			close(cvar)
			delete(isChannelOpen, cvar)
		}
	}
	isChannelOpenLock.Unlock()
endUnpause:
	//fmt.Printf("just unpaused actor %v", info)

	isActorPausedLock.Unlock()

	reply := Reply{StatusCode: http.StatusOK, ContentType: "application/json"}
	return json.Marshal(reply)

}

func unpauseNode(onlyUnpauseCurrent bool) ([]byte, error) {
	isNodePausedLock.Lock()
	if !onlyUnpauseCurrent {
		isNodePaused = false
		// clean up list of immune actors
		for key, _ := range immuneFromNodePaused {
			delete(immuneFromNodePaused, key)
		}
	}

	isActorPausedLock.Lock()
	//unpause whole node
	for actor, cvar := range isActorPaused {
		immuneFromNodePaused[actor] = struct{}{}
		isChannelOpenLock.Lock()
		isOpen, openOk := isChannelOpen[cvar]
		if openOk && isOpen {
			close(cvar)
			delete(isChannelOpen, cvar)
		}
		isChannelOpenLock.Unlock()
	}
	isActorPausedLock.Unlock()

	isNodePausedLock.Unlock()
	reply := Reply{StatusCode: http.StatusOK, ContentType: "application/json"}
	return json.Marshal(reply)
}


func informPause(info actorTuple_t, bk breakpoint_t, reqId string, flowId string, reqVal string, respVal string, isResponse bool){
	isResponseStr := "request"
	if isResponse { isResponseStr = "response" }
	var msg = map[string]string {
		"command": "notifyPause",
		"actorId": info.actorId,
		"actorType": info.actorType,
		"nodeId": rpc.GetNodeID(),
		"breakpointId": bk.id,
		"requestId": reqId,
		"flowId": flowId,
		"requestValue": reqVal,
		"responseValue": respVal,
		"isResponse": isResponseStr,
	}
	msgBytes, _ := json.Marshal(msg)

	informDebugger := func(debugger string){
		rpc.Tell(ctx, rpc.Destination{Target: rpc.Node{ID: debugger}, Method: sidecarEndpoint}, time.Time{}, "", msgBytes)
	}

	//inform all debuggers that we are paused
	debuggersMapLock.RLock()
	for debugger, _ := range debuggersMap {
		informDebugger(debugger)
	}
	debuggersMapLock.RUnlock()
}

func informBreakpoint(info actorTuple_t, reqId string, bk breakpoint_t, reqVal string, respVal string, isResponse string){
	var msg = map[string]string {
		"command": "notifyBreakpoint",
		"actorId": info.actorId,
		"actorType": info.actorType,
		"nodeId": rpc.GetNodeID(),
		"breakpointId": bk.id,
		"requestId": reqId,
		"requestValue": reqVal,
		"responseValue": respVal,
		"isResponse": isResponse,
	}
	msgBytes, _ := json.Marshal(msg)

	informDebugger := func(debugger string){
		rpc.Tell(ctx, rpc.Destination{Target: rpc.Node{ID: debugger}, Method: sidecarEndpoint}, time.Time{}, "", msgBytes)
	}

	//inform all debuggers that we hit a breakpoint
	debuggersMapLock.RLock()
	for debugger, _ := range debuggersMap {
		//fmt.Printf("Informing %v of breakpoint\n", debugger)
		informDebugger(debugger)
	}
	debuggersMapLock.RUnlock()
}

// wait on a paused actor
// note: condvar is pointer

func waitOnPause(info actorTuple_t, reqId string, flowId string, reqVal string, respVal string, isResponse bool, doInform bool){
	isResponseStr := "request"
	if isResponse { isResponseStr = "response" }
	isActorPausedLock.RLock()
	condvar, ok := isActorPaused[info]
	// assume that pausedBreaks[info] exists iff isActorPaused[info] does
	bk, _ := pausedBreaks[info]
	isActorPausedLock.RUnlock()

	if ok {
		//fmt.Printf("actor %v is rn waiting on pause\n", info)
		//fmt.Printf("\nImmune actors: %v\n", immuneFromNodePaused)
		informPause(info, bk, reqId, flowId, reqVal, respVal, isResponse)
		// wait on actor becoming unpaused

		waitingActorsLock.Lock()
		waitingActors[info] = waitInfo_t {
			ActorId: info.actorId,
			ActorType: info.actorType,
			FlowId: flowId,
			BreakpointId: bk.id,
			RequestId: reqId,
			RequestValue: reqVal,
			ResponseValue: respVal,
			IsResponse: isResponseStr,
		}
		waitingActorsLock.Unlock()

		<-condvar

		//fmt.Printf("actor %v rn woke up!\n", info)
		// woke up from wait — now unpaused

		waitingActorsLock.Lock()
		delete(waitingActors, info)
		waitingActorsLock.Unlock()


		isActorPausedLock.Lock()
		delete(isActorPaused, info)
		delete(pausedBreaks, info)
		//fmt.Printf("actor %v rn no longer waiting on pause\n", info)
		//fmt.Printf("paused rn actors: ")
		/*for key, _ := range isActorPaused {
			fmt.Printf("%v ", key)
		}*/
		isActorPausedLock.Unlock()
	}

	isActorPausedLock.RLock()
	nodeActor := actorTuple_t { actorId: "", actorType: ""}
	condvar, ok = isActorPaused[nodeActor]
	bk, _ = pausedBreaks[nodeActor]
	isActorPausedLock.RUnlock()

	if ok {
		//fmt.Printf("actor %v node-pause\n", info)
		informPause(info, bk, reqId, flowId, reqVal, respVal, isResponse)

		waitingActorsLock.Lock()
		waitingActors[info] = waitInfo_t {
			ActorId: info.actorId,
			ActorType: info.actorType,
			BreakpointId: bk.id,
			RequestId: reqId,
			FlowId: flowId,
			RequestValue: reqVal,
			ResponseValue: respVal,
			IsResponse: isResponseStr,
		}
		waitingActorsLock.Unlock()

		<-condvar
		// woke up from wait — now unpaused

		//fmt.Printf("actor %v woke up!\n", info)

		waitingActorsLock.Lock()
		delete(waitingActors, info)
		waitingActorsLock.Unlock()

		isActorPausedLock.Lock()
		delete(isActorPaused, nodeActor)
		delete(pausedBreaks, nodeActor)
		isActorPausedLock.Unlock()
		//fmt.Printf("actor %v no longer waiting on pause\n", info)

		waitingActorsLock.RLock()
		//fmt.Printf("waiting actors: %v", waitingActors)
		waitingActorsLock.RUnlock()
	}
}

// pause all sidecars
func pauseAllSidecars(bk breakpoint_t){
	var msg = map[string]string {
		"command": "pause",
		"actorType": "",
		"actorId": "",
		"breakpointId": bk.id,
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil { return }

	doTell := func(sidecar string){
		rpc.Tell(ctx, rpc.Destination{Target: rpc.Node{ID: sidecar}, Method: sidecarEndpoint}, time.Time{}, "", msgBytes)
	}

	sidecars, _ := rpc.GetNodeIDs()
	for _, sidecar := range sidecars {
		if sidecar != rpc.GetNodeID() {
			doTell(sidecar)
		}
	}
}

// register a node as a debugger

func registerDebugger(node string) ([]byte, error) {
	debuggersMapLock.Lock()
	defer debuggersMapLock.Unlock()
	debuggersMap[node] = true

	isDebuggerPresentLock.Lock()
	isDebuggerPresent = true
	//fmt.Println("Debuggers present " + node)
	isDebuggerPresentLock.Unlock()

	reply := Reply{StatusCode: http.StatusOK, ContentType: "application/json"}
	return json.Marshal(reply)
}

func unregisterDebugger(node string) ([]byte, error) {
	debuggersMapLock.Lock()
	defer debuggersMapLock.Unlock()
	delete(debuggersMap, node)
	if len(debuggersMap) == 0 {
		isDebuggerPresentLock.Lock()
		isDebuggerPresent = false

		if !config.IsDebugMode {
			// no debuggers present anymore
			// if we weren't started in debug mode, then unpause all actors, delete all breakpoints
			unpauseNode(false)

			breakpointsLock.Lock()
			breakpoints = map[string]breakpoint_t{}
			breakpointsByAttrs = map[breakpointAttrs_t]breakpoint_t{}
			breakpointsLock.Unlock()
		}

		isDebuggerPresentLock.Unlock()

		//fmt.Println("No debuggers present")
	} else {
		//fmt.Println(debuggersMap)
		//fmt.Println(node)
	}

	reply := Reply{StatusCode: http.StatusOK, ContentType: "application/json"}
	return json.Marshal(reply)
}

func listBreakpoints() ([]byte, error) {
	breakpointsLock.RLock()
	defer breakpointsLock.RUnlock()

	breakpointsMap := map[string]map[string]string {}

	for id, breakpoint := range breakpoints {
		breakpointsMap[id] = map[string]string {}
		breakpointsMap[id]["breakpointType"] = breakpoint.breakpointType
		breakpointsMap[id]["actorId"] = breakpoint.attrs.actorId
		breakpointsMap[id]["actorType"] = breakpoint.attrs.actorType
		breakpointsMap[id]["path"] = breakpoint.attrs.path
		breakpointsMap[id]["isRequest"] = breakpoint.attrs.isRequest
	}

	breakpointsBytes, err := json.Marshal(breakpointsMap)
	if err != nil { return nil, err }
	reply := Reply{StatusCode: http.StatusOK, ContentType: "application/json", Payload: string(breakpointsBytes) }
	return json.Marshal(reply)
}

func listPausedActors() ([]byte, error) {
	waitingActorsLock.RLock()
	defer waitingActorsLock.RUnlock()

	actorsList := []waitInfo_t {}

	for _, waitInfo := range waitingActors {
		//waitInfo.NodeId = rpc.GetNodeID()
		actorsList = append(actorsList, waitInfo)
	}

	waitBytes, err := json.Marshal(actorsList)
	if err != nil { return nil, err }
	reply := Reply{StatusCode: http.StatusOK, ContentType: "application/json", Payload: string(waitBytes) }
	return json.Marshal(reply)
}

func listBusyActors() ([]byte, error) {
	busyInfoLock.RLock()
	defer busyInfoLock.RUnlock()

	busyBytes, err := json.Marshal(busyInfo)
	if err != nil { return nil, err }
	reply := Reply{StatusCode: http.StatusOK, ContentType: "application/json", Payload: string(busyBytes) }
	return json.Marshal(reply)
}

// inform a debugger component attached to this sidecar that a node has
// been paused / a node's actor has been paused

func notifyPause(msg map[string]string) {
	myBytes, _ := json.Marshal(msg)
	sendAll(myBytes, "")
}

func notifyBreakpoint(msg map[string]string) {
	//fmt.Println("notifying breakpoint")
	myBytes, _ := json.Marshal(msg)
	sendAll(myBytes, "")
}

// activate an actor
func activate(ctx context.Context, actor Actor, session string, causingMsg map[string]string) ([]byte, error) {
	activatePath := actorRuntimeRoutePrefix + actor.Type + "/" + actor.ID + "?session=" + session
	reply, err := invoke(ctx, "GET", map[string]string{"path": activatePath}, actor.Type+":activate")
	if err != nil {
		if err != ctx.Err() {
			logger.Debug("activate failed to invoke %s: %v", actorRuntimeRoutePrefix+actor.Type+"/"+actor.ID, err)
		}
		return nil, err
	}
	if reply.StatusCode >= http.StatusBadRequest {
		if causingMsg["command"] == "call" {
			logger.Debug("activate %v returned status %v with body %s, aborting call %s", actor, reply.StatusCode, reply.Payload, causingMsg["path"])
		} else {
			// Log at error level becasue there is no one waiting on the method reponse to notice the failure.
			logger.Error("activate %v returned status %v with body %s, aborting tell %s", actor, reply.StatusCode, reply.Payload, causingMsg["path"])
		}
		return json.Marshal(reply)
	}
	logger.Debug("activate %v returned status %v with body %s", actor, reply.StatusCode, reply.Payload)
	return nil, nil
}

// invoke the deactivate method of an actor
func deactivate(ctx context.Context, actor *rpc.SessionInstance) {
	reply, err := invoke(ctx, "DELETE", map[string]string{"path": actorRuntimeRoutePrefix + actor.Name + "/" + actor.ID}, actor.Name+":deactivate")
	if err != nil {
		if err != ctx.Err() {
			logger.Debug("deactivate failed to invoke %s: %v", actorRuntimeRoutePrefix+actor.Name+"/"+actor.ID, err)
		}
		return
	}
	actor.Activated = false
	if reply.StatusCode >= http.StatusBadRequest {
		logger.Error("deactivate %v returned status %v with body %s", actor, reply.StatusCode, reply.Payload)
	} else {
		logger.Debug("deactivate %v returned status %v with body %s", actor, reply.StatusCode, reply.Payload)
	}
	return
}

func checkBreakpoint(attrs breakpointAttrs_t) (bool, breakpoint_t) {
	//fmt.Printf("Checking breakpoint %v\n", attrs)
	//fmt.Printf("\tCurrent breakpoints: %v\n", breakpointsByAttrs)
	breakpointsLock.Lock()
	defer breakpointsLock.Unlock()

	flowAttrs := breakpointAttrs_t {
		flowId: attrs.flowId,
	}

	bk, ok := breakpointsByAttrs[flowAttrs]
	if ok {
		if bk.deleteOnHit {
			go func() {
				retBytes, _ := implUnsetBreakpoint(map[string]string {
					"breakpointId": bk.id,
				})
				sendAll(retBytes, "")
			}()
		}
		return true, bk
	}

	attrs.flowId = ""
	bk, ok = breakpointsByAttrs[attrs]
	
	if ok {
		//fmt.Println(breakpointsByAttrs)
		if bk.deleteOnHit {
			go func() {
				retBytes, _ := implUnsetBreakpoint(map[string]string {
					"breakpointId": bk.id,
				})
				sendAll(retBytes, "")
			}()
			//delete(breakpointsByAttrs, attrs)
			//delete(breakpoints, bk.id)
		}
		return true, bk
	}

	newAttrs := attrs
	newAttrs.actorId = "" //check for wildcards on actorId
	bk, ok = breakpointsByAttrs[newAttrs]
	if ok {
		if bk.deleteOnHit {
			go func() {
				retBytes, _ := implUnsetBreakpoint(map[string]string {
					"breakpointId": bk.id,
				})
				sendAll(retBytes, "")
			}()
			//delete(breakpointsByAttrs, attrs)
			//delete(breakpoints, bk.id)
		}
		return true, bk
	}
	return false, breakpoint_t {}
}

////////////////////
// Misc. runtime operations
////////////////////

// Collect periodically collect actors with no recent usage (but retains placement)
func Collect(ctx context.Context) {
	if config.ActorCollectorInterval == 0 {
		logger.Info("Inactive actor collection disabled")
		return
	}
	lock := make(chan struct{}, 1) // trylock
	ticker := time.NewTicker(config.ActorCollectorInterval)
	for {
		select {
		case now := <-ticker.C:
			select {
			case lock <- struct{}{}:
				rpc.CollectInactiveSessions(ctx, now.Add(-config.ActorCollectorInterval), deactivate)
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
