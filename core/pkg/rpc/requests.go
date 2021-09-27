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

package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/IBM/kar/core/pkg/logger"
	"github.com/google/uuid"
)

var (
	requests = sync.Map{}           // map[string](chan Result){} // map request ids to result channels
	handlers = map[string]Handler{} // registered method handlers
)

func eval(ctx context.Context, method string, target Target, deadline time.Time, value []byte) ([]byte, string) {
	if !deadline.IsZero() && deadline.Before(time.Now()) {
		return nil, "deadline expired"
	}
	f := handlers[method]
	if f == nil {
		return nil, "undefined method " + method
	} else {
		result, err := f(ctx, target, value)
		if err != nil {
			b, _ := json.Marshal(err.Error) // attempt to serialize error object, ignore errors
			return b, err.Error()
		} else {
			return result, ""
		}
	}
}

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
		value, errMsg := eval(ctx, m.method(), m.target(), m.deadline(), m.value())
		err := Send(ctx, Response{RequestID: m.requestID(), Deadline: m.deadline(), Node: m.Caller, ErrMsg: errMsg, Value: value})
		if err != nil && err != ctx.Err() && err != ErrUnavailable {
			logger.Fatal("Producer error: cannot respond to call %s: %v", m.requestID(), err)
		}
	case TellRequest:
		_, errMsg := eval(ctx, m.method(), m.target(), m.deadline(), m.value())
		if errMsg != "" {
			logger.Warning("tell %s returned an error: %s", m.requestID(), errMsg)
		}
		if _, ok := m.target().(Node); !ok {
			err := Send(ctx, Done{RequestID: m.requestID(), Deadline: m.deadline()})
			if err != nil && err != ctx.Err() && err != ErrUnavailable {
				logger.Fatal("Producer error: cannot record completion for tell %s: %v", m.requestID(), err)
			}
		}
	}
}

// Call method and wait for result
func call(ctx context.Context, target Target, method string, deadline time.Time, value []byte) ([]byte, error) {
	requestID, ch, err := async(ctx, target, method, deadline, value)
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
func tell(ctx context.Context, target Target, method string, deadline time.Time, value []byte) error {
	requestID := uuid.New().String()
	return Send(ctx, TellRequest{RequestID: requestID, Target: target, Method: method, Deadline: deadline, Value: value})
}

// Call method and return a request id and a result channel
func async(ctx context.Context, target Target, method string, deadline time.Time, value []byte) (string, <-chan Result, error) {
	requestID := uuid.New().String()
	ch := make(chan Result, 1) // capacity one to be able to store result before accepting it
	requests.Store(requestID, ch)
	err := Send(ctx, CallRequest{RequestID: requestID, Target: target, Method: method, Deadline: deadline, Value: value})
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
func register(method string, handler Handler) {
	handlers[method] = handler
}

// Connect to Kafka
func connect(ctx context.Context, topic string, conf *Config, services ...string) (<-chan struct{}, error) {
	return Dial(ctx, topic, conf, services, func(msg Message) { go accept(ctx, msg) })
}