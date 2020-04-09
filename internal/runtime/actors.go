package runtime

import (
	"context"
	"sync"
	"time"

	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/internal/pubsub"
)

// Actor uniquely identifies an actor instance.
type Actor struct {
	Type string // actor type
	ID   string // actor instance id
}

type actorEntry struct {
	time    time.Time     // last release time
	lock    chan struct{} // entry lock
	valid   bool          // false iff entry has been removed from table
	session string        // current session or "" if none
	depth   int           // session depth
	busy    chan struct{} // close to notify end of session
}

var actorTable = sync.Map{} // actor table: Actor -> *actorEntry

// acquire locks the actor
// acquire returns nil if actor is mapped to another sidecar or context is cancelled
// acquire returns true if actor requires activation
func (actor Actor) acquire(ctx context.Context, session string) (*actorEntry, bool, error) {
	e := &actorEntry{lock: make(chan struct{}, 1)}
	for {
		if v, loaded := actorTable.LoadOrStore(actor, e); loaded {
			e = v.(*actorEntry) // existing entry
			select {
			case e.lock <- struct{}{}: // lock entry
			case <-ctx.Done():
				return nil, false, ctx.Err()
			}
			if e.valid {
				if e.session == session { // session is already in progress
					e.depth++
					<-e.lock
					return e, false, nil
				} else if e.session == "" { // start new session
					e.session = session
					e.depth = 1
					e.busy = make(chan struct{})
					<-e.lock
					return e, false, nil
				}
				// another session is in progress
				busy := e.busy // read while holding the lock
				<-e.lock
				select {
				case <-busy: // wait
				case <-ctx.Done():
					return nil, false, ctx.Err()
				}
				// loop around
				// no fairness issue trying to reacquire because we waited on busy
			} else {
				<-e.lock // invalid entry
				// loop around
				// no fairness issue trying to reacquire because this entry is dead
			}
		} else { // new entry
			e.lock <- struct{}{} // lock entry
			sidecar, err := pubsub.GetSidecar(actor.Type, actor.ID)
			if err != nil {
				<-e.lock
				return nil, false, err
			}
			if sidecar == config.ID { // start new session
				e.valid = true
				e.session = session
				e.depth = 1
				e.busy = make(chan struct{})
				<-e.lock
				return e, true, nil
			}
			actorTable.Delete(actor)
			<-e.lock // actor has been moved
			return nil, false, nil
		}
	}
}

// release updates the timestamp and releases the actor lock
func (e *actorEntry) release(ctx context.Context) {
	select {
	case e.lock <- struct{}{}: // lock entry
	case <-ctx.Done():
		return
	}
	e.depth--
	if e.depth == 0 { // end session
		e.session = ""
		close(e.busy)
	}
	e.time = time.Now() // update last release time
	<-e.lock
}

// collect deactivates actors that not been used since time
func collect(ctx context.Context, time time.Time) error {
	actorTable.Range(func(actor, v interface{}) bool {
		e := v.(*actorEntry)
		select {
		case e.lock <- struct{}{}:
			if e.valid && e.session == "" && e.time.Before(time) {
				deactivate(ctx, actor.(Actor))
				e.valid = false
				<-e.lock
				actorTable.Delete(actor)
			} else {
				<-e.lock
			}
		default:
		}
		return ctx.Err() == nil // stop collection if cancelled
	})
	return ctx.Err()
}

// Migrate deactivates actor if active and deletes placement if local
func Migrate(ctx context.Context, actor Actor) error {
	e, fresh, err := actor.acquire(ctx, "migration")
	if err != nil {
		return err
	}
	if e == nil {
		return nil
	}
	if !fresh {
		deactivate(ctx, actor)
	}
	select {
	case e.lock <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	close(e.busy)
	e.valid = false
	_, err = pubsub.CompareAndSetSidecar(actor.Type, actor.ID, config.ID, "") // delete placement if local
	actorTable.Delete(actor)
	<-e.lock
	return err
}
