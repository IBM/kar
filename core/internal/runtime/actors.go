package runtime

import (
	"context"
	"errors"
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
	actor   Actor
	time    time.Time     // last release time
	lock    chan struct{} // entry lock, never held for long, no need to watch ctx.Done()
	valid   bool          // false iff entry has been removed from table
	session string        // current session or "" if none
	depth   int           // session depth
	busy    chan struct{} // close to notify end of session
}

var (
	actorTable             = sync.Map{} // actor table: Actor -> *actorEntry
	errActorHasMoved       = errors.New("actor has moved")
	errActorAcquireTimeout = errors.New("timeout occured while acquiring actor")
)

// acquire locks the actor, session must be not be ""
// "exclusive" and "reminder" are reserved session names
// acquire returns true if actor requires activation before invocation
func (actor Actor) acquire(ctx context.Context, session string) (*actorEntry, bool, error) {
	e := &actorEntry{actor: actor, lock: make(chan struct{}, 1)}
	e.lock <- struct{}{} // lock entry
	for {
		if v, loaded := actorTable.LoadOrStore(actor, e); loaded {
			e := v.(*actorEntry) // found existing entry, := is required here!
			e.lock <- struct{}{} // lock entry
			if e.valid {
				if e.session == "" { // start new session
					e.session = session
					e.depth = 1
					e.busy = make(chan struct{})
					<-e.lock
					return e, false, nil
				}
				if session == "reminder" || session != "exclusive" && session == e.session { // reenter existing session
					e.depth++
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
				case <-time.After(config.ActorTimeout):
					return nil, false, errActorAcquireTimeout
				}
				// loop around
				// no fairness issue trying to reacquire because we waited on busy
			} else {
				<-e.lock // invalid entry
				// loop around
				// no fairness issue trying to reacquire because this entry is dead
			}
		} else { // new entry
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
			<-e.lock // actor has moved
			return nil, false, errActorHasMoved
		}
	}
}

// release releases the actor lock
// release updates the timestamp if the actor was invoked
// release removes the actor from the table if it was not activated at depth 0
func (e *actorEntry) release(session string, invoked bool) {
	e.lock <- struct{}{} // lock entry
	e.depth--
	if invoked {
		e.time = time.Now() // update last release time
	}
	if e.depth == 0 { // end session
		if !invoked { // actor was not activated
			e.valid = false
			actorTable.Delete(e.actor)
		}
		e.session = ""
		close(e.busy)
	}
	<-e.lock
}

// collect deactivates actors that not been used since time
func collect(ctx context.Context, time time.Time) error {
	actorTable.Range(func(actor, v interface{}) bool {
		e := v.(*actorEntry)
		select {
		case e.lock <- struct{}{}: // try acquire
			if e.valid && e.session == "" && e.time.Before(time) {
				e.depth = 1
				e.session = "exclusive"
				e.busy = make(chan struct{})
				<-e.lock
				err := deactivate(ctx, actor.(Actor))
				e.lock <- struct{}{}
				e.depth--
				e.session = ""
				if err == nil {
					e.valid = false
					actorTable.Delete(actor)
				}
				close(e.busy)
			}
			<-e.lock
		default:
		}
		return ctx.Err() == nil // stop collection if cancelled
	})
	return ctx.Err()
}

// migrate releases the actor lock and updates the sidecar for the actor
// the lock cannot be held multiple times
func (e *actorEntry) migrate(sidecar string) error {
	e.lock <- struct{}{}
	e.depth--
	e.session = ""
	e.valid = false
	actorTable.Delete(e.actor)
	_, err := pubsub.CompareAndSetSidecar(e.actor.Type, e.actor.ID, config.ID, sidecar)
	close(e.busy)
	<-e.lock
	return err
}
