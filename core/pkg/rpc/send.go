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

func alt(id string) string {
	return "alt_" + id
}

func instance(key string) (string, string) {
	parts := strings.Split(key, "_")
	return parts[1], parts[2]
}

// Lookup partition offering service
func routeToService(service string) (string, int32) {
	nodes := service2nodes[service]
	if len(nodes) == 0 {
		return "", 0
	}
	node := nodes[rand.Int31n(int32(len(nodes)))]
	return node, node2partition[node]
}

// Lookup partition offering session (errors: cancelled, Redis)
func routeToSession(ctx context.Context, service, session string) (string, int32, error) {
	nodes := service2nodes[service]
	if len(nodes) == 0 {
		return "", 0, nil // no matching service
	}

	key := place(service, session)
	if PlacementCache {
		if node, ok := session2NodeCache.Get(key); ok {
			return node.(string), node2partition[node.(string)], nil
		}
	}

	// Attempt to place (will discover global placement if already placed by someone else)
	node := ""
	next := nodes[rand.Int31n(int32(len(nodes)))] // select random node
	for ctx.Err() == nil {
		var err error
		node, err = store.CAS(ctx, key, node, next)
		if err != nil {
			return "", 0, err
		}
		partition := node2partition[node]
		if partition != 0 {
			if PlacementCache {
				session2NodeCache.Add(key, node)
			}
			return node, partition, nil
		}
	}
	return "", 0, ctx.Err()
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
	redirected := ""
	switch v := msg.(type) {
	case Request:
		switch t := v.target().(type) {
		case Service:
			_, partition = routeToService(t.Name)
		case Session:
			var err error
			_, partition, err = routeToSession(ctx, t.Name, t.ID)
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
			key := alt(v.requestID())
			node, _ := store.Get(ctx, key)
			if node == "" {
				return ErrUnavailable
			}
			redirected = key
			partition = node2partition[node]
		}
	case Done:
		// TODO use myPartition instead but resend message to partition 0 during recovery if request id occurs in partition 0
		partition = 0
	}

	// send message
	_, _, err := producer.SendMessage(encode(appTopic, partition, msg))
	if err == nil && redirected != "" {
		store.Del(ctx, redirected)
	}
	return err
}

// Resend request during recovery (errors: cancelled, Redis, Kafka)
func resend(ctx context.Context, msg Request, drop bool) error {
	// decide target partition
	var node string
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
		node, partition = routeToService(t.Name)
	case Session:
		var err error
		node, partition, err = routeToSession(ctx, t.Name, t.ID)
		if err != nil {
			return err
		}
	}
	if partition == 0 && drop {
		return nil
	}
	if msg.childID() != "" && node != "" {
		store.Set(ctx, alt(msg.childID()), node)
	}
	// resend message
	_, _, err := producer.SendMessage(encode(appTopic, partition, msg))
	return err
}
