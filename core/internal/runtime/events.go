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

package runtime

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/IBM/kar/core/internal/config"
	"github.com/IBM/kar/core/pkg/rpc"
)

// source describes an event source (subscription)
type source struct {
	// The actor that is subscribed to this source
	Actor Actor `json:"actor"`
	// The subscription id
	ID  string `json:"id"`
	key string // not serialized
	// The actor method that will be invoked to deliver the event to the actor
	Path string `json:"path"`
	// The topic that is the source of events for this subscription
	Topic string `json:"topic"`
	// The group ID for this consumer
	Group string `json:"group"`
	// The expected MIME type of events delivered by this subscription
	ContentType string `json:"contenttype,omitempty"`
	// Use the oldest available offset if no offset was previously committed
	OffsetOldest bool               `json:"oldestoffset"`
	cancel       context.CancelFunc // not serialized
	closed       <-chan struct{}    // not serialized
}

// EventSubscribeOptions documents the request body for subscribing an actor to a topic
type EventSubscribeOptions struct {
	// The expected MIME content type of the events that will be produced by this subscription
	// If an explicit value is not provided, the default value of application/json+cloudevent will be used.
	// Example: application/json
	ContentType string `json:"contentType,omitempty"`
	// The actor method to be invoked with each delivered event
	// Example: processEvent
	Path string `json:"path"`
	// The name of the topic being subscribed to
	Topic string `json:"topic"`
}

// topicCreateOptions documents the request body for creating a topic
type topicCreateOptions struct {
	NumPartitions     int32              `json:"numPartitions,omitempty"`
	ReplicationFactor int16              `json:"replicationFactor,omitempty"`
	ConfigEntries     map[string]*string `json:"configEntries,omitempty"`
}

func (s source) k() string {
	return s.key
}

// a collection of event sources
type sources map[Actor]map[string]source

var karPublisher rpc.Publisher

func init() {
	pairs["subscriptions"] = pair{bindings: sources{}, mu: &sync.Mutex{}}
}

// add binding to collection
func (c sources) add(ctx context.Context, b binding) (int, error) {
	s := b.(source)
	if _, ok := c[s.Actor]; !ok {
		c[s.Actor] = map[string]source{}
	}
	context, cancel := context.WithCancel(ctx)
	closed, code, err := subscribe(context, s)
	if err != nil {
		cancel()
		return code, err
	}
	s.cancel = cancel
	s.closed = closed
	c[s.Actor][s.ID] = s
	return http.StatusOK, nil
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

func subscribe(ctx context.Context, s source) (<-chan struct{}, int, error) {
	jsonType := s.ContentType == "" || // default is "application/cloudevents+json"
		s.ContentType == "text/json" ||
		s.ContentType == "application/json" ||
		strings.HasSuffix(s.ContentType, "+json")
	group := s.Group
	if group == "" {
		group = s.ID
	}

	rawEventToActorTellMsg := func(ctx context.Context, value []byte) ([]byte, error) {
		arg := string(value)

		// If the event is not already encoded as json, encode it as a json string
		if !jsonType {
			buf, err := json.Marshal(string(value))
			if err != nil {
				return nil, err
			}
			arg = string(buf)
		}

		// mirror command encoding from TellActor from commands.go
		msg := map[string]string{
			"command": "tell", // post with no callback expected
			"path":    s.Path,
			"payload": "[" + arg + "]"}

		return json.Marshal(msg)
	}

	ch, err := rpc.Subscribe(ctx, &config.KafkaConfig, s.Topic, group, s.OffsetOldest,
		rpc.Destination{Target: rpc.Session{Name: s.Actor.Type, ID: s.Actor.ID}, Method: actorEndpoint}, rawEventToActorTellMsg)

	if err == nil {
		return ch, http.StatusOK, nil
	} else if err == context.Canceled {
		return nil, http.StatusServiceUnavailable, err
	} else {
		return nil, http.StatusInternalServerError, err
	}
}
