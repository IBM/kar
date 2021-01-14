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

package runtime

import (
	"context"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.ibm.com/solsa/kar.git/core/internal/config"
	"github.ibm.com/solsa/kar.git/core/internal/pubsub"
	"github.ibm.com/solsa/kar.git/core/internal/store"
	"github.ibm.com/solsa/kar.git/core/pkg/logger"
)

// a persistent binding of an actor to something
type binding interface {
	k() string // cached redis key
}

// in-memory collection of bindings
// assumes exclusive access
type bindings interface {
	add(ctx context.Context, b binding) (int, error)
	cancel(actor Actor, id string) []binding
	find(actor Actor, id string) []binding

	// parse binding creation request payload to binding object and serialized binding (map[string]string)
	parse(actor Actor, id, key, payload string) (binding, map[string]string, error)

	// parse serialized binding
	load(actor Actor, id, key string, m map[string]string) (binding, error)
}

// a collection of bindings and a mutex to protect it
type pair struct {
	bindings bindings
	mu       *sync.Mutex
}

var (
	// binding kind -> binding collection
	pairs = map[string]pair{}
)

// redis key for binding
func bindingKey(kind string, actor Actor, partition, id string) string {
	return "binding" + config.Separator + partition + config.Separator + kind + config.Separator + actor.Type + config.Separator + actor.ID + config.Separator + id
}

// redis key for all bindings for a partition
func bindingPattern(partition string) string {
	return "binding" + config.Separator + partition + config.Separator + "*"
}

// binding for redis key
func keyBinding(key string) (kind string, actor Actor, partition, id string) {
	parts := strings.Split(key, config.Separator)
	partition = parts[1]
	kind = parts[2]
	actor = Actor{Type: parts[3], ID: parts[4]}
	id = parts[5]
	return
}

// load binding from redis in memory
func loadBinding(ctx context.Context, kind string, actor Actor, partition, id string) error {
	pair := pairs[kind]
	pair.mu.Lock()
	defer pair.mu.Unlock()
	key := bindingKey(kind, actor, partition, id)
	found := pair.bindings.find(actor, id)
	if len(found) > 0 { // bindingscription is already loaded
		return nil
	}
	data, err := store.HGetAll(key)
	if err != nil {
		return err
	}
	if len(data) == 0 { // bindingscription no longer exists
		return err
	}
	b, err := pair.bindings.load(actor, id, key, data)
	if err != nil {
		return err
	}
	_, err = pair.bindings.add(ctx, b)
	if err != nil {
		return err
	}
	logger.Debug("loaded binding %v", b)
	return nil
}

// delete bindings in redis and memory
func deleteBindings(kind string, actor Actor, id string) int {
	pair := pairs[kind]
	pair.mu.Lock()
	defer pair.mu.Unlock()
	found := pair.bindings.cancel(actor, id)
	for _, b := range found {
		store.Del(b.k())
	}
	logger.Debug("deleted %v binding(s) matching {%v, %v}", len(found), actor, id)
	return len(found)
}

// find bindings in memory
func getBindings(kind string, actor Actor, id string) []binding {
	pair := pairs[kind]
	pair.mu.Lock()
	defer pair.mu.Unlock()
	found := pair.bindings.find(actor, id)
	logger.Debug("found %v binding(s) matching {%v, %v}", len(found), actor, id)
	return found
}

// create or update a binding in redis and memory
func putBinding(ctx context.Context, kind string, actor Actor, id, payload string) (int, error) {
	pair := pairs[kind]
	pair.mu.Lock()
	defer pair.mu.Unlock()
	keys, _ := store.Keys(bindingKey(kind, actor, "*", id))
	var key string
	var successCode int
	if len(keys) > 0 { // reuse existing key
		key = keys[0]
		successCode = http.StatusOK
	} else { // new key with random partition
		ps, _ := pubsub.Partitions()
		p := ps[rand.Int31n(int32(len(ps)))]
		key = bindingKey(kind, actor, strconv.Itoa(int(p)), id)
		successCode = http.StatusNoContent
	}
	b, m, err := pair.bindings.parse(actor, id, key, payload)
	if err != nil {
		return http.StatusBadRequest, err
	}
	pair.bindings.cancel(actor, id)
	code, err := pair.bindings.add(ctx, b)
	if err != nil {
		return code, err
	}
	store.HSetMultiple(key, m)
	logger.Debug("put binding %v", b)
	return successCode, nil
}

// ensure bindings for this partition are loaded from redis in memory
func loadBindings(ctx context.Context, partitions []int32) error {
	logger.Debug("loadBindings starting")
	for _, p := range partitions {
		keys, err := store.Keys(bindingPattern(strconv.Itoa(int(p))))
		if err != nil {
			return err
		}
		logger.Debug("found %v persisted bindings for partition %v", len(keys), p)
		for _, key := range keys {
			kind, actor, partition, id := keyBinding(key)
			err := tellBinding(ctx, kind, actor, partition, id)
			if err != nil {
				if err != ctx.Err() {
					logger.Error("tell binding failed: %v", err)
				}
				return nil
			}
		}
	}
	logger.Debug("loadBindings completed")
	return nil
}
