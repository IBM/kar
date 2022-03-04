//
// Copyright IBM Corporation 2020,2022
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

	"github.com/IBM/kar/core/internal/config"
	"github.com/IBM/kar/core/pkg/logger"
	"github.com/IBM/kar/core/pkg/rpc"
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
	// If true, indicates that the Value represents a continuation
	TailCall bool `json:"tailCall,omitempty"`
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
	actor Actor
	time  time.Time     // last release time
	lock  chan struct{} // entry lock, never held for long, no need to watch ctx.Done()
	valid bool          // false iff entry has been removed from table
	flow  string        // current flow or "" if none
	depth int           // flow depth
	busy  chan struct{} // close to notify end of flow
	msg   map[string]string
}

var (
	actorTable             = sync.Map{} // actor table: Actor -> *actorEntry
	errActorHasMoved       = errors.New("actor has moved")
	errActorAcquireTimeout = errors.New("timeout occurred while acquiring actor")
)

// acquire locks the actor, flow must be not be ""
// "exclusive" and "nonexclusive" are reserved flow names
// acquire returns true if actor requires activation before invocation
func (actor Actor) acquire(ctx context.Context, flow string, msg map[string]string) (*actorEntry, bool, error, map[string]string) {
	e := &actorEntry{actor: actor, lock: make(chan struct{}, 1)}
	e.lock <- struct{}{} // lock entry
	for {
		if v, loaded := actorTable.LoadOrStore(actor, e); loaded {
			e := v.(*actorEntry) // found existing entry, := is required here!
			e.lock <- struct{}{} // lock entry
			if e.valid {
				if e.flow == "" { // start new flow
					e.flow = flow
					e.depth = 1
					e.busy = make(chan struct{})
					e.msg = msg
					<-e.lock
					return e, false, nil, nil
				}
				if flow == "nonexclusive" || flow != "exclusive" && flow == e.flow { // reenter existing flow
					if msg["lockRetained"] != "true" { // do not increment depth for tail call (lock was not released at end of previous step)
						e.depth++
					}
					<-e.lock
					return e, false, nil, nil
				}
				// another flow is in progress
				busy := e.busy // read while holding the lock
				<-e.lock
				if config.ActorBusyTimeout > 0 {
					select {
					case <-busy: // wait
					case <-ctx.Done():
						return nil, false, ctx.Err(), nil
					case <-time.After(config.ActorBusyTimeout):
						e.lock <- struct{}{}
						reason := e.msg
						<-e.lock
						return nil, false, errActorAcquireTimeout, reason
					}
				} else {
					select {
					case <-busy: // wait
					case <-ctx.Done():
						return nil, false, ctx.Err(), nil
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
			sidecar, err := rpc.GetSessionNodeID(ctx, rpc.Session{Name: actor.Type, ID: actor.ID})
			if err != nil {
				<-e.lock
				return nil, false, err, nil
			}
			if sidecar == rpc.GetNodeID() { // start new flow
				e.valid = true
				e.flow = flow
				e.depth = 1
				e.msg = msg
				e.busy = make(chan struct{})
				<-e.lock
				return e, true, nil, nil
			}
			actorTable.Delete(actor)
			<-e.lock // actor has moved
			return nil, false, errActorHasMoved, nil
		}
	}
}

// release releases the actor lock
// release updates the timestamp if the actor was invoked
// release removes the actor from the table if it was not activated at depth 0
func (e *actorEntry) release(flow string, invoked bool) {
	e.lock <- struct{}{} // lock entry
	e.depth--
	if invoked {
		e.time = time.Now() // update last release time
	}
	if e.depth == 0 { // end flow
		if !invoked { // actor was not activated
			e.valid = false
			actorTable.Delete(e.actor)
		}
		e.flow = ""
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
			if e.valid && e.flow == "" && e.time.Before(time) {
				e.depth = 1
				e.flow = "exclusive"
				e.busy = make(chan struct{})
				<-e.lock
				err := deactivate(ctx, actor.(Actor))
				e.lock <- struct{}{}
				e.depth--
				e.flow = ""
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
	sidecars, _ := rpc.GetNodeIDs()
	for _, sidecar := range sidecars {
		var actorInformation map[string][]string
		if sidecar != rpc.GetNodeID() {
			// Make call to another sidecar, returns the result of GetMyActiveActors() there
			msg := map[string]string{
				"command":   "getActiveActors",
				"actorType": targetedActorType,
			}
			bytes, err := json.Marshal(msg)
			if err != nil {
				logger.Debug("Error marshalling a map[string][string]: %v", err)
			}
			bytes, err = rpc.Call(ctx, rpc.Destination{Target: rpc.Node{ID: sidecar}, Method: sidecarEndpoint}, time.Time{}, bytes)
			if err != nil {
				logger.Debug("Error gathering actor information: %v", err)
				return nil, err
			}
			var actorReply Reply
			err = json.Unmarshal(bytes, &actorReply)
			if err != nil {
				logger.Debug("Error gathering actor information: %v", err)
				return nil, err
			}
			if actorReply.StatusCode != 200 {
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

// delete releases the actor lock and removes the placement for the actor
// the lock cannot be held multiple times
func (e *actorEntry) delete() error {
	e.lock <- struct{}{}
	e.depth--
	e.flow = ""
	e.valid = false
	actorTable.Delete(e.actor)
	err := rpc.DelSession(ctx, rpc.Session{Name: e.actor.Type, ID: e.actor.ID})
	close(e.busy)
	<-e.lock
	return err
}
