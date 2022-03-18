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
	"fmt"
	"sort"
	"strings"
	"time"

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
			bytes, err = rpc.Call(ctx, rpc.Destination{Target: rpc.Node{ID: sidecar}, Method: sidecarEndpoint}, time.Time{}, "", bytes)
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
			actorInformation = rpc.GetLocalActivatedSessions(ctx, targetedActorType)
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
