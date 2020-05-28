package runtime

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.ibm.com/solsa/kar.git/internal/pubsub"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

type source struct {
	Actor        Actor
	ID           string
	key          string // not serialized
	Path         string
	Topic        string
	Group        string
	ContentType  string
	OffsetOldest bool
	cancel       context.CancelFunc // not serialized
}

func (s source) k() string {
	return s.key
}

// TODO lock
// TODO synchronous subscribe and unsubscribe

// a collection of event sources
type sources map[Actor]map[string]source

func init() {
	pairs["subscriptions"] = pair{bindings: sources{}, mu: &sync.Mutex{}}
}

// add binding to collection
func (c sources) add(ctx context.Context, b binding) error {
	s := b.(source)
	if _, ok := c[s.Actor]; !ok {
		c[s.Actor] = map[string]source{}
	}
	ctx, s.cancel = context.WithCancel(ctx)
	err := subscribe(ctx, s)
	if err != nil {
		return err
	}
	c[s.Actor][s.ID] = s
	return nil
}

// find bindings in collection
func (c sources) find(actor Actor, id string) []binding {
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
func (c sources) cancel(actor Actor, id string) []binding {
	if id != "" {
		if b, ok := c[actor][id]; ok {
			b.cancel()
			delete(c[actor], id)
			return []binding{b}
		}
		return []binding{}
	}
	a := []binding{}
	for _, b := range c[actor] {
		b.cancel()
		a = append(a, b)
	}
	delete(c, actor)
	return a
}

func (c sources) parse(actor Actor, id, key, payload string) (binding, map[string]string, error) {
	var m map[string]string
	if err := json.Unmarshal([]byte(payload), &m); err != nil {
		return nil, nil, err
	}
	b, _ := c.load(actor, id, key, m)
	return b, m, nil
}

func (c sources) load(actor Actor, id, key string, m map[string]string) (binding, error) {
	return source{
		Actor:        actor,
		ID:           id,
		key:          key,
		Path:         m["path"],
		Topic:        m["topic"],
		Group:        m["group"],
		ContentType:  m["contentType"],
		OffsetOldest: m["offsetOldest"] == "true",
	}, nil
}

func subscribe(ctx context.Context, s source) error {
	contentType := s.ContentType
	if contentType == "" {
		contentType = "application/cloudevents+json"
	}
	group := s.Group
	if group == "" {
		group = s.ID
	}
	ch, err := pubsub.Subscribe(ctx, s.Topic, group, &pubsub.Options{OffsetOldest: s.OffsetOldest})
	if err != nil {
		return err
	}

	go func() {
		for msg := range ch {
			reply, err := CallActor(ctx, s.Actor, s.Path, string(msg.Value), contentType, "", "")
			msg.Mark()
			if err != nil {
				logger.Error("failed to post event to %s: %v", s.Path, err)
			} else {
				if reply.StatusCode >= http.StatusBadRequest {
					logger.Error("subscriber returned status %v with body %s", reply.StatusCode, reply.Payload)
				} else {
					logger.Debug("subscriber returned status %v with body %s", reply.StatusCode, reply.Payload)
				}
			}
		}
	}()

	return nil
}