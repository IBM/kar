package placement

import (
	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/internal/store"
)

func mangle(t, id string) string {
	return "placement" + config.Separator + t + config.Separator + id
}

// Get returns current sidecar for actor
func Get(t, id string) (string, error) {
	s, err := store.Get(mangle(t, id))
	if err == store.ErrNil {
		return "", nil
	}
	return s, err
}

// Update atomically updates current sidecar for actor (use empty string for no sidecar)
func Update(t, id, old, new string) (int, error) {
	return store.CompareAndSet(mangle(t, id), old, new)
}
