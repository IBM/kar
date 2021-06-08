//
// Copyright IBM Corporation 2020,2021
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/IBM/kar.git/core/internal/config"
	"github.com/IBM/kar.git/core/internal/pubsub"
	"github.com/IBM/kar.git/core/pkg/logger"
)

// Actor uniquely identifies an actor instance.
type Actor struct {
	Type string // actor type
	ID   string // actor instance id
}

// actorCallResult encodes the result of invoking an actor method.
type actorCallResult struct {
	// The value being returned as the result of the method
	Value interface{} `json:"value,omitempty"`
	// If true, indicates that the method execution resulted in an error/exception
	Error bool `json:"error,omitempty"`
	// When error is true, the error message
	Message string `json:"message,omitempty"`
	// When error is true, the stack trace for the error
	Stack string `json:"stack,omitempty"`
}

// stateUpdateOp describes a multi-element update operation on an Actors state
type stateUpdateOp struct {
	Updates        map[string]interface{}            `json:"updates,omitempty"`
	SubmapUpdates  map[string]map[string]interface{} `json:"submapupdates,omitempty"`
	Removals       []string                          `json:"removals,omitempty"`
	SubmapRemovals map[string][]string               `json:"submapremovals,omitempty"`
}

// submapOp describes the requested operation on a submap in an Actors state
type submapOp struct {
	Op string `json:"op"`
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
	errActorAcquireTimeout = errors.New("timeout occurred while acquiring actor")
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
				if config.ActorBusyTimeout > 0 {
					select {
					case <-busy: // wait
					case <-ctx.Done():
						return nil, false, ctx.Err()
					case <-time.After(config.ActorBusyTimeout):
						return nil, false, errActorAcquireTimeout
					}
				} else {
					select {
					case <-busy: // wait
					case <-ctx.Done():
						return nil, false, ctx.Err()
					}
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

// getMyActiveActors returns a map of actor types ->  list of active IDs in this sidecar
func getMyActiveActors(targetedActorType string) map[string][]string {
	information := make(map[string][]string)
	actorTable.Range(func(actor, v interface{}) bool {
		e := v.(*actorEntry)
		e.lock <- struct{}{}
		if e.valid {
			if targetedActorType == "" || targetedActorType == e.actor.Type {
				information[e.actor.Type] = append(information[e.actor.Type], e.actor.ID)
			}
		}
		<-e.lock
		return true
	})
	return information
}

// getAllActiveActors Returns map of actor types ->  list of active IDs for all sidecars in the app
func getAllActiveActors(ctx context.Context, targetedActorType string) (map[string][]string, error) {
	information := make(map[string][]string)
	for _, sidecar := range pubsub.Sidecars() {
		var actorInformation map[string][]string
		if sidecar != config.ID {
			// Make call to another sidecar, returns the result of GetMyActiveActors() there
			msg := map[string]string{
				"protocol":  "sidecar",
				"sidecar":   sidecar,
				"command":   "getActiveActors",
				"actorType": targetedActorType,
			}
			actorReply, err := callHelper(ctx, msg, false)
			if err != nil || actorReply.StatusCode != 200 {
				logger.Debug("Error gathering actor information: %v", err)
				return nil, err
			}
			err = json.Unmarshal([]byte(actorReply.Payload), &actorInformation)
			if err != nil {
				logger.Debug("Error unmarshaling actor information: %v", err)
				return nil, err
			}
		} else {
			actorInformation = getMyActiveActors(targetedActorType)
		}
		for actorType, actorIDs := range actorInformation { // accumulate sidecar's info into information
			information[actorType] = append(information[actorType], actorIDs...)
		}
	}
	return information, nil
}

func formatActorInstanceMap(actorInfo map[string][]string, format string) (string, error) {
	if format == "json" || format == "application/json" {
		var m []byte
		m, err := json.MarshalIndent(actorInfo, "", "  ")
		if err != nil {
			logger.Debug("Error marshaling actors information: %v", err)
			return "", err
		}
		return string(m), nil
	}
	var str strings.Builder
	for actorType, actorIDs := range actorInfo {
		sort.Strings(actorIDs)
		fmt.Fprintf(&str, "%v: [\n", actorType)
		for _, actorID := range actorIDs {
			fmt.Fprintf(&str, "    %v\n", actorID)
		}
		fmt.Fprintf(&str, "]\n")
	}
	return str.String(), nil
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
