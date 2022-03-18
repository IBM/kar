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

package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/IBM/kar/core/pkg/logger"
	"github.com/google/uuid"
)

type sessionKey struct {
	Name string
	ID   string
}

var (
	requests                         = sync.Map{}                  // map[string](chan Result){} // map request ids to result channels
	handlersService                  = map[string]ServiceHandler{} // registered method handlers for service targets
	handlersSession                  = map[string]SessionHandler{} // registered method handlers for session targets
	handlersNode                     = map[string]NodeHandler{}    // registered method handlers for node targets
	sessionTable                     = sync.Map{}                  // session table: SessionKey -> *SessionInstance
	deferredLocks                    = sync.Map{}                  // locks being defered by tail calls: uuid -> chan
	sessionBusyTimeout time.Duration = 0
)

func getLocalActivatedSessions(ctxt context.Context, name string) map[string][]string {
	information := make(map[string][]string)
	sessionTable.Range(func(key, v interface{}) bool {
		instance := v.(*SessionInstance)
		instance.lock <- struct{}{}
		if instance.valid && instance.Activated {
			if name == "" || instance.Name == name {
				information[instance.Name] = append(information[instance.Name], instance.ID)
			}
		}
		<-instance.lock
		return true
	})
	return information
}

func sendOrDie(ctx context.Context, msg Message) {
	err := Send(ctx, msg)
	if err != nil && err != ctx.Err() && err != ErrUnavailable {
		logger.Fatal("Producer error: cannot send message with request id %s: %v", msg.requestID(), err)
	}
}

// accept must not block; it is executing on the primary go routine that is receiving messages
func accept(ctx context.Context, msg Message) {
	switch m := msg.(type) {
	case Response:
		obj, ok := requests.LoadAndDelete(m.RequestID)
		if !ok {
			return // ignore responses without matching requests
		}
		ch := obj.(chan Result)
		result := Result{Value: m.Value}
		if m.ErrMsg != "" {
			result.Err = errors.New(m.ErrMsg)
		}
		ch <- result

	case CallRequest:
		// TODO remove waiting here once actor queue is implemented
		if m.childID() != "" {
			ch := make(chan Result, 1) // capacity one to be able to store result before accepting it
			requests.Store(m.childID(), ch)
			select {
			case <-ch:
			case <-ctx.Done():
				return
			}
			requests.Delete(m.childID())
		}

		if !m.deadline().IsZero() && m.deadline().Before(time.Now()) {
			go func() {
				errMsg := fmt.Sprintf("deadline expired: deadline was %v and it is now %v", m.deadline(), time.Now())
				sendOrDie(ctx, Response{RequestID: m.requestID(), Deadline: m.deadline(), Node: m.Caller, ErrMsg: errMsg, Value: nil})
			}()
			return
		}

		switch target := m.target().(type) {
		case Service:
			go func() {
				f := handlersService[m.method()]
				if f == nil {
					errMsg := fmt.Sprintf("undefined method %v", m.method())
					sendOrDie(ctx, Response{RequestID: m.requestID(), Deadline: m.deadline(), Node: m.Caller, ErrMsg: errMsg, Value: nil})
				} else {
					value, err := f(ctx, target, m.value())
					if err != nil {
						value, _ = json.Marshal(err) // attempt to serialize error object, ignore errors
						sendOrDie(ctx, Response{RequestID: m.requestID(), Deadline: m.deadline(), Node: m.Caller, ErrMsg: err.Error(), Value: value})
					} else {
						sendOrDie(ctx, Response{RequestID: m.requestID(), Deadline: m.deadline(), Node: m.Caller, ErrMsg: "", Value: value})
					}
				}
			}()

		case Session:
			acceptSession(ctx, target, m)

		case Node:
			go func() {
				f := handlersNode[m.method()]
				if f == nil {
					errMsg := fmt.Sprintf("undefined method %v", m.method())
					sendOrDie(ctx, Response{RequestID: m.requestID(), Deadline: m.deadline(), Node: m.Caller, ErrMsg: errMsg, Value: nil})
				} else {
					value, err := f(ctx, target, m.value())
					if err != nil {
						value, _ = json.Marshal(err) // attempt to serialize error object, ignore errors
						sendOrDie(ctx, Response{RequestID: m.requestID(), Deadline: m.deadline(), Node: m.Caller, ErrMsg: err.Error(), Value: value})
					} else {
						sendOrDie(ctx, Response{RequestID: m.requestID(), Deadline: m.deadline(), Node: m.Caller, ErrMsg: "", Value: value})
					}
				}
			}()
		}

	case TellRequest:
		// TODO remove waiting here once actor queue is implemented
		if m.childID() != "" {
			ch := make(chan Result, 1) // capacity one to be able to store result before accepting it
			requests.Store(m.childID(), ch)
			select {
			case <-ch:
			case <-ctx.Done():
				return
			}
			requests.Delete(m.childID())
		}

		if !m.deadline().IsZero() && m.deadline().Before(time.Now()) {
			go func() {
				logger.Warning("tell %s to %v dropped at time %v due to expired deadline %v", m.requestID(), m.target(), time.Now(), m.deadline())
				sendOrDie(ctx, Done{RequestID: m.requestID(), Deadline: m.deadline()})
			}()
			return
		}

		switch target := m.target().(type) {
		case Service:
			go func() {
				f := handlersService[m.method()]
				if f == nil {
					logger.Warning("tell %s to %v requested undefined method %v", m.requestID(), m.target(), m.method())
					sendOrDie(ctx, Done{RequestID: m.requestID(), Deadline: m.deadline()})
				} else {
					_, err := f(ctx, target, m.value())
					if err != nil && err != ctx.Err() {
						logger.Warning("tell %s to %v returned an error: %v", m.requestID(), m.target(), err)
					}
					sendOrDie(ctx, Done{RequestID: m.requestID(), Deadline: m.deadline()})
				}
			}()

		case Session:
			acceptSession(ctx, target, m)

		case Node:
			// No matter what happens we don't need to send a Done record; a Node-targeted TellRequest is not replayed on failure.
			go func() {
				f := handlersNode[m.method()]
				if f == nil {
					logger.Warning("tell %s to %v requested undefined method %v", m.requestID(), m.target(), m.method())
				} else {
					_, err := f(ctx, target, m.value())
					if err != nil && err != ctx.Err() {
						logger.Warning("tell %s to %v returned an error: %v", m.requestID(), m.target(), err)
					}
				}
			}()
		}
	}
}

// acceptSession must not block; it is executing on the primary go routine that is receiving messages
func acceptSession(ctx context.Context, target Session, msg Request) {
	// Step 1: Get valid SessionInstance for target Session
	key := sessionKey{Name: target.Name, ID: target.ID}
	var instance *SessionInstance = nil
	freshInstance := false
	if v, ok := sessionTable.Load(key); ok {
		e := v.(*SessionInstance)
		e.lock <- struct{}{} // lock entry
		if e.valid {
			instance = e
		} else {
			<-e.lock // unlock invalid entry; we won't use it again but there may be multiple deactivate tasks queued up
		}
	}
	if instance == nil {
		instance = &SessionInstance{Name: target.Name, ID: target.ID, ActiveFlow: target.Flow, next: make(chan struct{}, 1), lock: make(chan struct{}, 1), valid: true}
		instance.next <- struct{}{} // enable tasks
		instance.lock <- struct{}{} // lock entry
		freshInstance = true
		sessionTable.Store(key, instance)
	}

	// Step 2: Schedule the go-routine that will actually do the processing of the msg
	if !freshInstance && instance.ActiveFlow == target.Flow {
		// re-entrancy bypass; handler must execute "concurrently" with ancestors
		var dl chan struct{} = nil
		if target.DeferredLockID != "" {
			if l, ok := deferredLocks.LoadAndDelete(target.DeferredLockID); ok {
				dl = l.(chan struct{})
			}
		}
		if dl == nil {
			logger.Debug("reentrant message %v for %v; no deferred lock", msg.requestID(), target)
		} else {
			logger.Debug("reentrant message %v for %v; found deferred lock", msg.requestID(), target)
		}

		go handleSessionRequest(ctx, nil, dl, instance, target, msg)
	} else if target.Flow == "nonexclusive" {
		logger.Debug("nonexclusive message for %v", target)
		go handleSessionRequest(ctx, nil, nil, instance, target, msg)
	} else {
		before := instance.next
		instance.next = make(chan struct{}, 1)
		logger.Debug("queued message for %v", target)
		go handleSessionRequest(ctx, before, instance.next, instance, target, msg)
	}

	// Step 3: Release lock
	<-instance.lock // unlock entry
}

// handleSessionRequest executes on a go routine spawned to process a single request; it can safely block
func handleSessionRequest(ctx context.Context, before chan struct{}, after chan struct{}, instance *SessionInstance, target Session, m Request) {
	if before != nil {
		// wait for my turn to execute
		logger.Debug("%v is waiting to execute %v", target, m.requestID())
		if sessionBusyTimeout > 0 {
			select {
			case <-before:
			case <-ctx.Done():
				return
			case <-time.After(sessionBusyTimeout):
				errMsg := fmt.Sprintf("Possible deadlock: timed out waiting in instance queue for %v", target)
				if cr, ok := m.(CallRequest); ok {
					sendOrDie(ctx, Response{RequestID: cr.requestID(), Deadline: cr.deadline(), Node: cr.Caller, ErrMsg: errMsg, Value: nil})
				} else {
					logger.Warning(errMsg)
					sendOrDie(ctx, Done{RequestID: m.requestID(), Deadline: m.deadline()})
				}
				// The actual task timed out, but I am still responsible for releasing `after` at the appropriate time
				if after != nil {
					select {
					case <-before:
						after <- struct{}{}
						return
					case <-ctx.Done():
						return
					}
				}
			}
		} else {
			// Simple case.  No timeout, so just wait for my turn
			select {
			case <-before:
			case <-ctx.Done():
				return
			}
		}
	}

	// Now it is my turn to execute.
	logger.Debug("%v is executing %v", target, m.requestID())
	if target.Flow != "nonexclusive" {
		instance.ActiveFlow = target.Flow
	}
	f := handlersSession[m.method()]
	if f == nil {
		errMsg := fmt.Sprintf("undefined method %v", m.method())
		if cr, ok := m.(CallRequest); ok {
			sendOrDie(ctx, Response{RequestID: m.requestID(), Deadline: m.deadline(), Node: cr.Caller, ErrMsg: errMsg, Value: nil})
		} else {
			logger.Warning(errMsg)
			sendOrDie(ctx, Done{RequestID: m.requestID(), Deadline: m.deadline()})
		}
	} else {
		dest, value, err := f(ctx, target, instance, m.requestID(), m.value()) // The call to the higher-level handler that does something useful....at last!!!
		if instance.Activated && target.Flow != "nonexclusive" {
			instance.lastAccess = time.Now()
		}

		if err != nil {
			if err != ctx.Err() {
				if cr, ok := m.(CallRequest); ok {
					value, _ = json.Marshal(err) // attempt to serialize error object, ignore errors
					sendOrDie(ctx, Response{RequestID: m.requestID(), Deadline: m.deadline(), Node: cr.Caller, ErrMsg: err.Error(), Value: value})
				} else {
					logger.Warning("tell %s to %v returned an error: %v", m.requestID(), m.target(), err)
					sendOrDie(ctx, Done{RequestID: m.requestID(), Deadline: m.deadline()})
				}
			}
		} else {
			if dest == nil {
				if cr, ok := m.(CallRequest); ok {
					sendOrDie(ctx, Response{RequestID: m.requestID(), Deadline: m.deadline(), Node: cr.Caller, ErrMsg: "", Value: value})
				} else {
					sendOrDie(ctx, Done{RequestID: m.requestID(), Deadline: m.deadline()})
				}
			} else {
				if next, ok := dest.Target.(Session); ok && after != nil && next.DeferredLockID != "" {
					// Defer my obligation to release after to the next invocation of this flow on this instance
					logger.Debug("%v executing %v is deferring lock to %v", target, m.requestID(), next.DeferredLockID)
					if next.Flow != instance.ActiveFlow {
						logger.Error("Improper lock deferal from flow %v to flow %v", instance.ActiveFlow, next.Flow)
					}
					deferredLocks.Store(next.DeferredLockID, after)
					after = nil
				}
				if cr, ok := m.(CallRequest); ok {
					sendOrDie(ctx, CallRequest{RequestID: m.requestID(), Deadline: m.deadline(), Caller: cr.Caller, Value: value, Target: dest.Target, Method: dest.Method, Sequence: cr.Sequence + 1})
				} else {
					tr := m.(TellRequest)
					sendOrDie(ctx, TellRequest{RequestID: m.requestID(), Deadline: m.deadline(), Value: value, Target: dest.Target, Method: dest.Method, Sequence: tr.Sequence + 1})
				}
			}
		}
	}

	// Finally, if I am responsible for releasing the next task, do so.
	if after != nil {
		logger.Debug("%v executing %v released lock", target, m.requestID())
		after <- struct{}{}
	}
}

// Call method and wait for result
func call(ctx context.Context, dest Destination, deadline time.Time, parentID string, value []byte) ([]byte, error) {
	requestID, ch, err := async(ctx, dest, deadline, parentID, value)
	if err != nil {
		return nil, err
	}
	defer requests.Delete(requestID)
	select {
	case result := <-ch:
		return result.Value, result.Err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Call method and return immediately (result will be discarded)
func tell(ctx context.Context, dest Destination, deadline time.Time, value []byte) error {
	requestID := uuid.New().String()
	return Send(ctx, TellRequest{RequestID: requestID, Target: dest.Target, Method: dest.Method, Deadline: deadline, Value: value})
}

// Call method and return a request id and a result channel
func async(ctx context.Context, dest Destination, deadline time.Time, parentID string, value []byte) (string, <-chan Result, error) {
	requestID := uuid.New().String()
	ch := make(chan Result, 1) // capacity one to be able to store result before accepting it
	requests.Store(requestID, ch)
	err := Send(ctx, CallRequest{RequestID: requestID, Target: dest.Target, Method: dest.Method, Deadline: deadline, Value: value, ParentID: parentID})
	if err != nil {
		requests.Delete(requestID)
		return "", nil, err
	}
	return requestID, ch, nil
}

// Reclaim resources associated with request id
func reclaim(requestID string) {
	requests.Delete(requestID)
}

// Register method handler
func registerService(method string, handler ServiceHandler) {
	handlersService[method] = handler
}

func registerSession(method string, handler SessionHandler) {
	handlersSession[method] = handler
}

func registerNode(method string, handler NodeHandler) {
	handlersNode[method] = handler
}

// Connect to Kafka
func connect(ctx context.Context, topic string, conf *Config, services ...string) (<-chan struct{}, error) {
	sessionBusyTimeout = conf.SessionBusyTimeout
	return Dial(ctx, topic, conf, services, func(msg Message) { accept(ctx, msg) })
}
