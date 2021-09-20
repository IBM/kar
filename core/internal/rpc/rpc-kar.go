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

/////////////////
// Porting code to incrementally adust to the new APIs
// When we are done, this file will be empty
/////////////////

package rpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Staging types to allow migration to new RPC library
type KarHandler func(context.Context, Target, []byte) ([]byte, error)

////////
// Staging code...these methods are meant to be directly replacable by their corresponding RPC versions once the APIs converge
////////

func RegisterKAR(method string, handler KarHandler) {
	handlers[method] = handler
}

// CallPromiseKAR makes a call via pubsub to a sidecar and returns a promise that may later be used to await the reply
func CallPromiseKAR(ctx context.Context, target Target, method string, value []byte) (string, error) {
	request := uuid.New().String()
	ch := make(chan *[]byte)
	requests.Store(request, ch)
	// defer requests.Delete(request)
	err := send(ctx, target, method, karCallbackInfo{SendingNode: getNodeID(), Request: request}, value)
	if err != nil {
		return "", err
	}
	return request, nil
}

// AwaitPromiseKAR awaits the response to an actor or service call
func AwaitPromiseKAR(ctx context.Context, request string) ([]byte, error) {
	if ch, ok := requests.Load(request); ok {
		defer requests.Delete(request)
		select {
		case r := <-ch.(chan *[]byte):
			if r != nil {
				return *r, nil
			} else {
				return []byte{}, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, fmt.Errorf("unexpected request %s", request)
}
