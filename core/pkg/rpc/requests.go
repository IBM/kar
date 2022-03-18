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

var (
	requests        = sync.Map{}                  // map[string](chan Result){} // map request ids to result channels
	handlersService = map[string]ServiceHandler{} // registered method handlers for service targets
	handlersSession = map[string]SessionHandler{} // registered method handlers for session targets
	handlersNode    = map[string]NodeHandler{}    // registered method handlers for node targets
)

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
			go func() {
				f := handlersSession[m.method()]
				if f == nil {
					errMsg := fmt.Sprintf("undefined method %v", m.method())
					sendOrDie(ctx, Response{RequestID: m.requestID(), Deadline: m.deadline(), Node: m.Caller, ErrMsg: errMsg, Value: nil})
				} else {
					dest, value, err := f(ctx, target, m.requestID(), m.value())
					if err != nil {
						value, _ = json.Marshal(err) // attempt to serialize error object, ignore errors
						sendOrDie(ctx, Response{RequestID: m.requestID(), Deadline: m.deadline(), Node: m.Caller, ErrMsg: err.Error(), Value: value})
					} else {
						if dest == nil {
							sendOrDie(ctx, Response{RequestID: m.requestID(), Deadline: m.deadline(), Node: m.Caller, ErrMsg: "", Value: value})
						} else {
							sendOrDie(ctx, CallRequest{RequestID: m.requestID(), Deadline: m.deadline(), Caller: m.Caller, Value: value, Target: dest.Target, Method: dest.Method, Sequence: m.Sequence + 1})
						}
					}
				}
			}()

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
			go func() {
				f := handlersSession[m.method()]
				if f == nil {
					logger.Warning("tell %s to %v requested undefined method %v", m.requestID(), m.target(), m.method())
					sendOrDie(ctx, Done{RequestID: m.requestID(), Deadline: m.deadline()})
				} else {
					dest, value, err := f(ctx, target, m.requestID(), m.value())
					if err != nil && err != ctx.Err() {
						logger.Warning("tell %s to %v returned an error: %v", m.requestID(), m.target(), err)
						sendOrDie(ctx, Done{RequestID: m.requestID(), Deadline: m.deadline()})
					} else {
						if dest == nil {
							sendOrDie(ctx, Done{RequestID: m.requestID(), Deadline: m.deadline()})
						} else {
							sendOrDie(ctx, TellRequest{RequestID: m.requestID(), Deadline: m.deadline(), Value: value, Target: dest.Target, Method: dest.Method, Sequence: m.Sequence + 1})
						}
					}
				}
			}()

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
	return Dial(ctx, topic, conf, services, func(msg Message) { accept(ctx, msg) })
}
