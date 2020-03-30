package runtime

import (
	"context"
	"sync"
	"time"

	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/internal/pubsub"
	"golang.org/x/sync/semaphore"
)

// Actor uniquely identifies an actor instance.
type Actor struct {
	Type string // actor type
	ID   string // actor instance id
}

type actorEntry struct {
	time  time.Time           // last release time
	sem   *semaphore.Weighted // cancellable trylock, held while actor is in use
	valid bool                // false iff entry has been removed from table
}

var actorTable = sync.Map{} // actor table: Actor -> *actorEntry

// acquire locks the actor
// acquire returns nil if actor is mapped to another sidecar or context is cancelled
// acquire returns true if actor requires activation
func (actor Actor) acquire(ctx context.Context) (*actorEntry, bool, error) {
	e := &actorEntry{sem: semaphore.NewWeighted(1)}
	for {
		if v, loaded := actorTable.LoadOrStore(actor, e); loaded {
			e = v.(*actorEntry) // existing entry
			err := e.sem.Acquire(ctx, 1)
			if err != nil { // cancelled
				return nil, false, err
			}
			if e.valid {
				return e, false, nil
			}
			e.sem.Release(1) // invalid entry, try again
		} else { // new entry
			err := e.sem.Acquire(ctx, 1)
			if err != nil { // cancelled
				return nil, false, err
			}
			sidecar, err := pubsub.GetSidecar(actor.Type, actor.ID)
			if err != nil {
				return nil, false, err
			}
			if sidecar == config.ID {
				e.valid = true
				return e, true, nil
			}
			e.sem.Release(1) // actor has been moved
			actorTable.Delete(actor)
			return nil, false, nil
		}
	}
}

// release updates the timestamp and releases the actor lock
func (e *actorEntry) release() {
	e.time = time.Now() // update last release time
	e.sem.Release(1)
}

// collect deactivates actors that not been used since time
func collect(ctx context.Context, time time.Time) error {
	actorTable.Range(func(actor, v interface{}) bool {
		e := v.(*actorEntry)
		if e.sem.TryAcquire(1) { // skip actor if busy
			if e.valid && e.time.Before(time) {
				deactivate(ctx, actor.(Actor))
				e.valid = false
				e.sem.Release(1)
				actorTable.Delete(actor)
			} else {
				e.sem.Release(1)
			}
		}
		return ctx.Err() == nil // stop collection if cancelled
	})
	return ctx.Err()
}

// Migrate deactivates actor if active and deletes placement if local
func Migrate(ctx context.Context, actor Actor) error {
	e, fresh, err := actor.acquire(ctx)
	if err != nil {
		return err
	}
	if e == nil {
		return nil
	}
	if !fresh {
		deactivate(ctx, actor)
	}
	e.valid = false
	_, err = pubsub.CompareAndSetSidecar(actor.Type, actor.ID, config.ID, "") // delete placement if local
	e.sem.Release(1)
	actorTable.Delete(actor)
	return err
}
