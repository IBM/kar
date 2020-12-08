package pubsub

import (
	"strings"

	"github.ibm.com/solsa/kar.git/core/internal/config"
	"github.ibm.com/solsa/kar.git/core/internal/store"
)

func placementKeyPrefix(t string) string {
	return "pubsub" + config.Separator + "placement" + config.Separator + t
}

func placementKey(t, id string) string {
	return "pubsub" + config.Separator + "placement" + config.Separator + t + config.Separator + id
}

// GetSidecar returns the current sidecar for the given actor type and id or "" if none.
func GetSidecar(t, id string) (string, error) {
	s, err := store.Get(placementKey(t, id))
	if err == store.ErrNil {
		return "", nil
	}
	return s, err
}

// CompareAndSetSidecar atomically updates the sidecar for the given actor type and id.
// Use old = "" to atomically set the initial placement.
// Use new = "" to atomically delete the current placement.
// Returns 0 if unsuccessful, 1 if successful.
func CompareAndSetSidecar(t, id, old, new string) (int, error) {
	o := &old
	if old == "" {
		o = nil
	}
	n := &new
	if new == "" {
		n = nil
	}
	return store.CompareAndSet(placementKey(t, id), o, n)
}

// GetAllActorInstances returns a mapping from actor types to instanceIDs
func GetAllActorInstances(actorTypePrefix string) (map[string][]string, error) {
	m := map[string][]string{}
	reply, err := store.Keys(placementKeyPrefix(actorTypePrefix) + "*")
	if err != nil {
		return nil, err
	}
	for _, key := range reply {
		splitKeys := strings.Split(key, config.Separator)
		actorType := splitKeys[2]
		instanceID := splitKeys[3]
		if m[actorType] == nil {
			m[actorType] = make([]string, 0)
		}
		m[actorType] = append(m[actorType], instanceID)
	}
	return m, nil
}
