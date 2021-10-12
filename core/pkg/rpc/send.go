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
	"math/rand"
	"strings"

	"github.com/IBM/kar/core/pkg/logger"
	"github.com/IBM/kar/core/pkg/store"
)

func place(names ...string) string {
	key := "rpc"
	for _, name := range names {
		key += "_" + name
	}
	return key
}

func instance(key string) (string, string) {
	parts := strings.Split(key, "_")
	return parts[1], parts[2]
}

// Lookup partition offering service
func routeToService(service string) int32 {
	nodes := service2nodes[service]
	if len(nodes) == 0 {
		return 0
	}
	return node2partition[nodes[rand.Int31n(int32(len(nodes)))]]
}

// Lookup partition offering session (errors: cancelled, Redis)
func routeToSession(ctx context.Context, service, session string) (int32, error) {
	nodes := service2nodes[service]
	if len(nodes) == 0 {
		return 0, nil // no matching service
	}
	node := ""
	next := nodes[rand.Int31n(int32(len(nodes)))] // select random node
	for ctx.Err() == nil {
		var err error
		node, err = store.CAS(ctx, place(service, session), node, next)
		if err != nil {
			return 0, err
		}
		partition := node2partition[node]
		if partition != 0 {
			return partition, nil
		}
	}
	return 0, ctx.Err()
}

// Send message (errors: cancelled, Redis, Kafka, ErrUnavailable)
func Send(ctx context.Context, msg Message) error {
	// acquire R mutex
	mu.RLock()
	defer mu.RUnlock()

	// return if context is cancelled
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// decide target partition
	var partition int32
	switch v := msg.(type) {
	case Request:
		switch t := v.target().(type) {
		case Service:
			partition = routeToService(t.Name)
		case Session:
			var err error
			partition, err = routeToSession(ctx, t.Name, t.ID)
			if err != nil {
				return err
			}
		case Node:
			partition = node2partition[t.ID]
			if partition == 0 {
				return ErrUnavailable
			}
		}
	case Response:
		partition = node2partition[v.Node]
		if partition == 0 {
			return ErrUnavailable
		}
	case Done:
		// TODO use myPartition instead but resend message to partition 0 during recovery if request id occurs in partition 0
		partition = 0
	}

	// send message
	_, _, err := producer.SendMessage(encode(appTopic, partition, msg))
	return err
}

// Resend request during recovery (errors: cancelled, Redis, Kafka)
func resend(ctx context.Context, msg Request, drop bool) error {
	// decide target partition
	var partition int32
	switch t := msg.target().(type) {
	case Node:
		switch v := msg.(type) {
		case CallRequest:
			m := Response{RequestID: v.RequestID, Node: v.Caller, ErrMsg: "node died before processing call request", Value: nil}
			_, _, err := producer.SendMessage(encode(appTopic, node2partition[v.Caller], m))
			return err
		case TellRequest:
			logger.Warning("node died before processing tell request with id %s", v.requestID())
			return nil
		}
	case Service:
		partition = routeToService(t.Name)
	case Session:
		var err error
		partition, err = routeToSession(ctx, t.Name, t.ID)
		if err != nil {
			return err
		}
	}
	if partition == 0 && drop {
		return nil
	}

	// resend message
	_, _, err := producer.SendMessage(encode(appTopic, partition, msg))
	return err
}
