package runtime

import (
	"context"
	"encoding/json"
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
	closed       <-chan struct{}
}

func (s source) k() string {
	return s.key
}

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
	context, cancel := context.WithCancel(ctx)
	closed, err := subscribe(context, s)
	if err != nil {
		cancel()
		return err
	}
	s.cancel = cancel
	s.closed = closed
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
			<-b.closed
			delete(c[actor], id)
			return []binding{b}
		}
		return []binding{}
	}
	a := []binding{}
	for _, b := range c[actor] {
		b.cancel()
		<-b.closed
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

func subscribe(ctx context.Context, s source) (<-chan struct{}, error) {
	contentType := s.ContentType
	if contentType == "" {
		contentType = "application/cloudevents+json"
	}
	group := s.Group
	if group == "" {
		group = s.ID
	}

	f := func(msg pubsub.Message) {
		err := TellActor(ctx, s.Actor, s.Path, "["+string(msg.Value)+"]", contentType, "POST")
		if err != nil {
			logger.Error("failed to post event to %s: %v", s.Path, err)
		} else {
			msg.Mark()
		}
	}

	return pubsub.Subscribe(ctx, s.Topic, group, &pubsub.Options{OffsetOldest: s.OffsetOldest}, f)
}
