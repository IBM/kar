package actors

import (
	"context"
	"sync"
	"time"

	"github.ibm.com/solsa/kar.git/pkg/logger"
	"golang.org/x/sync/semaphore"
)

var (
	table = map[string]*Entry{} // actor table
	lock  = sync.Mutex{}        // table lock
)

// Entry is the type of table entries
type Entry struct {
	time  time.Time           // last release time
	sem   *semaphore.Weighted // cancellable trylock, held while actor is in use
	valid bool                // false iff entry has been removed from table
}

// Activate acquires the entry initializing the entry if absent
func Activate(ctx context.Context, key string) (*Entry, bool) {
	for {
		lock.Lock()
		if e, ok := table[key]; ok {
			lock.Unlock()
			err := e.sem.Acquire(ctx, 1)
			if err != nil { // cancelled
				return nil, false
			}
			if e.valid {
				return e, false // existing entry
			}
			e.sem.Release(1) // deleted, try again
		} else {
			e = &Entry{sem: semaphore.NewWeighted(1), valid: true}
			e.sem.Acquire(ctx, 1) // no risk of failure
			table[key] = e
			lock.Unlock()
			return e, true // new entry
		}
	}
}

// Unlock updates the timestamp and unlocks the entry
func (e *Entry) Unlock() {
	e.time = time.Now() // update last release time
	e.sem.Release(1)
}

// get tries to acquire the entry if already present
func get(key string) *Entry {
	for {
		lock.Lock()
		if e, ok := table[key]; ok {
			lock.Unlock()
			if !e.sem.TryAcquire(1) {
				return nil // entry is busy, abort
			}
			if e.valid {
				return e // key already present
			}
			e.sem.Release(1) // deleted, try again
		} else {
			lock.Unlock()
			return nil // key absent
		}
	}
}

// deactivate releases and deletes the entry
func (e *Entry) deactivate(key string) {
	lock.Lock()
	delete(table, key)
	lock.Unlock()
	e.valid = false
	e.sem.Release(1)
}

// Collect removes entries older than time
func Collect(time time.Time, f func(key string)) {
	for key := range table { // TODO parallel loop?
		if e := get(key); e != nil {
			if e.time.Before(time) {
				logger.Debug("deactivating entry %s", key)
				f(key)
				e.deactivate(key)
			} else {
				e.Unlock()
			}
		}
	}
}
