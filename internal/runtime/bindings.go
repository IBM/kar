package runtime

import (
	"context"
	"math/rand"
	"strconv"
	"strings"
	"sync"

	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/internal/pubsub"
	"github.ibm.com/solsa/kar.git/internal/store"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

// a persistent binding of an actor to something
type binding interface {
	k() string // cached redis key
}

type base struct {
	Actor   Actor
	ID      string
	key     string
	Payload string
}

func (s base) k() string {
	return s.key
}

// in-memory collection of bindings
// assumes exclusive access
type bindings interface {
	add(b binding)
	cancel(actor Actor, id string) []binding
	find(actor Actor, id string) []binding
	parse(actor Actor, id, key, payload string) (binding, map[string]string, error)
	load(actor Actor, id, key string, m map[string]string) (binding, error)
}

// a collection of bindings implemented as a map
type collection map[Actor]map[string]binding

// add binding to collection
func (c collection) add(b binding) {
	x := b.(base)
	if _, ok := c[x.Actor]; !ok {
		c[x.Actor] = map[string]binding{}
	}
	c[x.Actor][x.ID] = b
}

// find bindings in collection
func (c collection) find(actor Actor, id string) []binding {
	if id != "" {
		if b, ok := c[actor][id]; ok {
			return []binding{b}
		}
		return []binding{}
	}
	a := []binding{}
	for _, b := range c[actor] {
		a = append(a, b)
	}
	return a
}

// remove bindings from collection
func (c collection) cancel(actor Actor, id string) []binding {
	if id != "" {
		if b, ok := c[actor][id]; ok {
			delete(c[actor], id)
			return []binding{b}
		}
		return []binding{}
	}
	a := []binding{}
	for _, b := range c[actor] {
		a = append(a, b)
	}
	delete(c, actor)
	return a
}

func (c collection) parse(actor Actor, id, key, payload string) (binding, map[string]string, error) {
	return base{Actor: actor, ID: id, key: key, Payload: payload}, map[string]string{"payload": payload}, nil
}

func (c collection) load(actor Actor, id, key string, m map[string]string) (binding, error) {
	return base{Actor: actor, ID: id, key: key, Payload: m["payload"]}, nil
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

func init() {
	pairs["subscriptions"] = pair{bindings: collection{}, mu: &sync.Mutex{}}
	pairs["reminders"] = pair{bindings: &activeReminders, mu: &sync.Mutex{}}
}

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
func loadBinding(kind string, actor Actor, partition, id string) error {
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
	pair.bindings.add(b)
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

// add binding in redis and memory
func postBinding(kind string, actor Actor, id, payload string) error {
	pair := pairs[kind]
	pair.mu.Lock()
	defer pair.mu.Unlock()
	keys, _ := store.Keys(bindingKey(kind, actor, "*", id))
	var key string
	if len(keys) > 0 { // reuse existing key
		key = keys[0]
	} else { // new key with random partition
		ps, _ := pubsub.Partitions()
		p := ps[rand.Int31n(int32(len(ps)))]
		key = bindingKey(kind, actor, strconv.Itoa(int(p)), id)
	}
	b, m, err := pair.bindings.parse(actor, id, key, payload)
	if err != nil {
		return err
	}
	pair.bindings.cancel(actor, id)
	pair.bindings.add(b)
	store.HSetMultiple(key, m)
	logger.Debug("created binding %v", b)
	return nil
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
