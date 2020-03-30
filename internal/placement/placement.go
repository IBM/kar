// Package placement persists the placement of actors onto sidecars.
package placement

import (
	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/internal/store"
)

func mangle(t, id string) string {
	return "placement" + config.Separator + t + config.Separator + id
}

// Get returns the current sidecar for the given actor type and id or "" if none.
func Get(t, id string) (string, error) {
	s, err := store.Get(mangle(t, id))
	if err == store.ErrNil {
		return "", nil
	}
	return s, err
}

// CompareAndSet atomically updates the sidecar for the given actor type and id.
// Use old = "" to atomically set the initial placement.
// Use new = "" to atomically delete the current placement.
func CompareAndSet(t, id, old, new string) (int, error) {
	o := &old
	if old == "" {
		o = nil
	}
	n := &new
	if new == "" {
		n = nil
	}
	return store.CompareAndSet(mangle(t, id), o, n)
}
