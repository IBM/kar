package actors

import (
	"context"
	"sync"
	"time"

	"github.ibm.com/solsa/kar.git/pkg/logger"
	"golang.org/x/sync/semaphore"
)

var table = sync.Map{} // actor table

// Entry is the type of table entries
type Entry struct {
	time  time.Time           // last release time
	sem   *semaphore.Weighted // cancellable trylock, held while actor is in use
	valid bool                // false iff entry has been removed from table
}

// Acquire acquires the entry initializing the entry if absent
func Acquire(ctx context.Context, key string) (*Entry, bool) {
	e := &Entry{sem: semaphore.NewWeighted(1)}
	for {
		if v, loaded := table.LoadOrStore(key, e); loaded {
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
func Collect(ctx context.Context, time time.Time, f func(key string)) {
	table.Range(func(key, v interface{}) bool {
		e := v.(*Entry)
		if e.sem.TryAcquire(1) {
			if e.valid && e.time.Before(time) {
				logger.Debug("deactivating %s", key)
				f(key.(string))
				e.valid = false
				e.sem.Release(1)
				table.Delete(key)
			} else {
				e.sem.Release(1)
			}
		}
		return ctx.Err() == nil // stop collection if cancelled
	})
}
