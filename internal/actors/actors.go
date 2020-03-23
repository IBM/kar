package actors

import (
	"context"
	"sync"
	"time"

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
		} else {
			err := e.sem.Acquire(ctx, 1)
			if err != nil { // cancelled
				return nil, false
			}
			e.valid = true
			return e, true
		}
	}
}

// Release updates the timestamp and releases the entry
func (e *Entry) Release() {
	e.time = time.Now() // update last release time
	e.sem.Release(1)
}

// Collect removes entries older than time
func Collect(ctx context.Context, time time.Time, f func(actor Actor)) {
	table.Range(func(actor, v interface{}) bool {
		e := v.(*Entry)
		if e.sem.TryAcquire(1) {
			if e.valid && e.time.Before(time) {
				f(actor.(Actor))
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
