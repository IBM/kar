package runtime

import "sync"

type source struct {
	Actor   Actor
	ID      string
	key     string
	Payload string
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
func (c sources) add(b binding) {
	s := b.(source)
	if _, ok := c[s.Actor]; !ok {
		c[s.Actor] = map[string]source{}
	}
	c[s.Actor][s.ID] = s
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

func (c sources) parse(actor Actor, id, key, payload string) (binding, map[string]string, error) {
	return source{Actor: actor, ID: id, key: key, Payload: payload}, map[string]string{"payload": payload}, nil
}

func (c sources) load(actor Actor, id, key string, m map[string]string) (binding, error) {
	return source{Actor: actor, ID: id, key: key, Payload: m["payload"]}, nil
}
