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
	"sort"

	"github.com/IBM/kar/core/pkg/store"
)

func getNodeID() string {
	return self.Node
}

func getTopology() (map[string][]string, <-chan struct{}) {
	mu.RLock()
	defer mu.RUnlock()
	node2services := map[string][]string{}
	for service, nodes := range service2nodes {
		for _, node := range nodes {
			node2services[node] = append(node2services[node], service)
		}
	}
	return node2services, tick
}

func getNodeIDs() ([]string, <-chan struct{}) {
	mu.RLock()
	defer mu.RUnlock()
	nodes := make([]string, len(node2partition))
	i := 0
	for p := range node2partition {
		nodes[i] = p
		i++
	}
	sort.Strings(nodes)
	return nodes, tick
}

func getServices() ([]string, <-chan struct{}) {
	mu.RLock()
	defer mu.RUnlock()
	services := make([]string, len(service2nodes))
	i := 0
	for s := range service2nodes {
		services[i] = s
		i++
	}
	sort.Strings(services)
	return services, tick
}

func getServiceNodeIDs(service string) ([]string, <-chan struct{}) {
	mu.RLock()
	defer mu.RUnlock()
	nodes := make([]string, len(service2nodes[service]))
	copy(nodes, service2nodes[service]) // copy array before sorting in place
	sort.Strings(nodes)
	return nodes, tick
}

func getPartition() int32 {
	return self.Partition
}

func getPartitions() ([]int32, <-chan struct{}) {
	mu.RLock()
	defer mu.RUnlock()
	return []int32{0}, tick // TODO: fix hack
	/*
		partitions := make([]int32, len(node2partition))
		i := 0
		for _, p := range node2partition {
			partitions[i] = p
			i++
		}
		sort.Slice(partitions, func(i, j int) bool { return partitions[i] < partitions[j] })
		return partitions, tick
	*/
}

func getSessionNodeID(ctx context.Context, session Session) (string, error) {
	node, err := store.Get(ctx, place(session.Name, session.ID))
	if err == store.ErrNil {
		err = nil
	}
	return node, err
}

func delSession(ctx context.Context, session Session) error {
	_, err := store.Del(ctx, place(session.Name, session.ID))
	return err
}

func getAllSessions(ctx context.Context, sessionPrefixFilter string) (map[string][]string, error) {
	pattern := place("*")
	if sessionPrefixFilter != "" {
		pattern = place(sessionPrefixFilter, "*")
	}
	m := map[string][]string{}
	reply, err := store.Keys(ctx, pattern)
	if err != nil {
		return nil, err
	}
	for _, key := range reply {
		actorType, instanceID := instance(key)
		if m[actorType] == nil {
			m[actorType] = make([]string, 0)
		}
		m[actorType] = append(m[actorType], instanceID)
	}
	return m, nil
}
