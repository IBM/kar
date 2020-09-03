package pubsub

import (
	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/internal/store"
)

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
