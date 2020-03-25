package actors

import (
	"context"
	"sync"
	"time"

	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/internal/store"
	"golang.org/x/sync/semaphore"
)

// Actor uniquely identifies an actor instance
type Actor struct {
	Type string // actor type
	ID   string // actor instance id
}

// Entry is the type of table entries
type Entry struct {
	time  time.Time           // last release time
	sem   *semaphore.Weighted // cancellable trylock, held while actor is in use
	valid bool                // false iff entry has been removed from table
}

var table = sync.Map{} // actor table: ID -> Entry

// Acquire acquires the entry initializing the entry if absent
func Acquire(ctx context.Context, actor Actor) (*Entry, bool) {
	e := &Entry{sem: semaphore.NewWeighted(1)}
	for {
		if v, loaded := table.LoadOrStore(actor, e); loaded {
			e = v.(*Entry) // existing entry
			err := e.sem.Acquire(ctx, 1)
			if err != nil { // cancelled
				return nil, false
			}
			if e.valid {
				return e, false
			}
			e.sem.Release(1) // deleted, try again
		} else { // new entry
			err := e.sem.Acquire(ctx, 1)
			if err != nil { // cancelled
				return nil, false
			}
			if sidecar, _ := Get(actor); sidecar == config.ID { // check placement
				e.valid = true
				return e, true
			}
			e.sem.Release(1) // actor has been moved
			table.Delete(actor)
			return nil, false
		}
	}
}

// Release updates the timestamp and releases the entry
func (e *Entry) Release() {
	e.time = time.Now() // update last release time
	e.sem.Release(1)
}

// Collect removes entries older than time
func Collect(ctx context.Context, time time.Time, deactivate func(actor Actor)) {
	table.Range(func(actor, v interface{}) bool {
		e := v.(*Entry)
		if e.sem.TryAcquire(1) {
			if e.valid && e.time.Before(time) {
				deactivate(actor.(Actor))
				e.valid = false
				e.sem.Release(1)
				table.Delete(actor)
			} else {
				e.sem.Release(1)
			}
		}
		return ctx.Err() == nil // stop collection if cancelled
	})
}

// Migrate deactivates actor if active and deletes placement
func Migrate(ctx context.Context, actor Actor, deactivate func(actor Actor)) error {
	e := &Entry{sem: semaphore.NewWeighted(1)}
	for {
		if v, loaded := table.LoadOrStore(actor, e); loaded {
			e = v.(*Entry) // existing entry
			err := e.sem.Acquire(ctx, 1)
			if err != nil { // cancelled
				return err
			}
			if e.valid {
				deactivate(actor)
				e.valid = false
				Update(actor, config.ID, "") // delete placement if placed here
				e.sem.Release(1)
				table.Delete(actor)
				return nil
			}
			e.sem.Release(1) // deleted, try again
		} else { // new entry
			err := e.sem.Acquire(ctx, 1)
			if err != nil { // cancelled
				return err
			}
			Update(actor, config.ID, "") // delete placement if placed here
			e.sem.Release(1)
			table.Delete(actor)
			return nil
		}
	}
}

func mangle(actor Actor) string {
	return "actors" + config.Separator + "sidecar" + config.Separator + actor.Type + config.Separator + actor.ID
}

// Get returns current sidecar for actor
func Get(actor Actor) (string, error) {
	return store.Get(mangle(actor))
}

// Update atomically updates current sidecar for actor (use empty string for no sidecar)
func Update(actor Actor, old, new string) (int, error) {
	return store.CompareAndSet(mangle(actor), old, new)
}
