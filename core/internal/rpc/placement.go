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
	"strings"

	"github.com/IBM/kar/core/pkg/store"
)

// separator character for store keys and topic names
const separator = "_" // must not be a legal DNS name character; private copy to avoid circular dependency

func placementKeyPrefix(t string) string {
	return "pubsub" + separator + "placement" + separator + t
}

func placementKey(t, id string) string {
	return "pubsub" + separator + "placement" + separator + t + separator + id
}

// getSidecar returns the current sidecar for the given actor type and id or "" if none.
func getSidecar(ctx context.Context, t, id string) (string, error) {
	s, err := store.Get(ctx, placementKey(t, id))
	if err == store.ErrNil {
		return "", nil
	}
	return s, err
}

// compareAndSetSidecar atomically updates the sidecar for the given actor type and id.
// Use old = "" to atomically set the initial placement.
// Use new = "" to atomically delete the current placement.
// Returns 0 if unsuccessful, 1 if successful.
func compareAndSetSidecar(ctx context.Context, t, id, old, new string) (int, error) {
	o := &old
	if old == "" {
		o = nil
	}
	n := &new
	if new == "" {
		n = nil
	}
	return store.CompareAndSet(ctx, placementKey(t, id), o, n)
}

// getAllSessions returns a mapping from actor types to instanceIDs
func getAllSessions(ctx context.Context, actorTypePrefix string) (map[string][]string, error) {
	m := map[string][]string{}
	reply, err := store.Keys(ctx, placementKeyPrefix(actorTypePrefix)+"*")
	if err != nil {
		return nil, err
	}
	for _, key := range reply {
		splitKeys := strings.Split(key, separator)
		actorType := splitKeys[2]
		instanceID := splitKeys[3]
		if m[actorType] == nil {
			m[actorType] = make([]string, 0)
		}
		m[actorType] = append(m[actorType], instanceID)
	}
	return m, nil
}
