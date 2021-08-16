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

/*
 * This file contains the implementation of the portion of the
 * KAR REST API related to actor state.
 */

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/IBM/kar/core/internal/config"
	"github.com/IBM/kar/core/internal/store"
	"github.com/julienschmidt/httprouter"
)

func stateKey(t, id string) string {
	return "main" + config.Separator + "state" + config.Separator + t + config.Separator + id
}

func flatEntryKey(key string) string {
	return key + config.Separator + config.Separator
}

func nestedEntryKey(key string, subkey string) string {
	return key + config.Separator + subkey
}

func nestedEntryKeyPrefix(key string) string {
	return key + config.Separator
}

// swagger:route HEAD /v1/actor/{actorType}/{actorId}/state/{key} state idActorStateExists
//
// state/key
//
// ### Check to see if single entry of an actor's state is defined
//
// Check to see if the state of the actor instance indicated by `actorType` and `actorId`
// contains an entry for `key`.
//
//     Consumes:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200StateExistsResult
//       404: response404
//       500: response500
//

// swagger:route HEAD /v1/actor/{actorType}/{actorId}/state/{key}/{subkey} state idActorStateSubkeyExists
//
// state/key/subkey
//
// ### Check to see if single entry of an actor's state is defined
//
// Check to see if the state of the actor instance indicated by `actorType` and `actorId`
// contains an entry for `key`/`subkey`.
//
//     Consumes:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200StateExistsResult
//       404: response404
//       500: response500
//
func routeImplContainsKey(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var mangledEntryKey string
	if subkey := ps.ByName("subkey"); subkey != "" {
		mangledEntryKey = nestedEntryKey(ps.ByName("key"), subkey)
	} else {
		mangledEntryKey = flatEntryKey(ps.ByName("key"))
	}

	if reply, err := store.HExists(stateKey(ps.ByName("type"), ps.ByName("id")), mangledEntryKey); err != nil {
		http.Error(w, fmt.Sprintf("HExists failed: %v", err), http.StatusInternalServerError)
	} else {
		if reply == 0 {
			http.Error(w, "key not present", http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}
}

// swagger:route PUT /v1/actor/{actorType}/{actorId}/state/{key} state idActorStateSet
//
// state/key
//
// ### Update a single entry of an actor's state
//
// The state of the actor instance indicated by `actorType` and `actorId`
// will be updated by setting `key` to contain the JSON request body.
// The operation will not return until the state has been updated.
//
//     Consumes:
//     - application/json
//     Produces:
//     - text/plain
//     Schemes: http
//     Responses:
//       201: response201
//       204: response204
//       500: response500
//

// swagger:route PUT /v1/actor/{actorType}/{actorId}/state/{key}/{subkey} state idActorStateSubkeySet
//
// state/key/subkey
//
// ### Update a single entry of a sub-map of an actor's state
//
// The map state of the actor instance indicated by `actorType` and `actorId`
// will be updated by setting `key`/`subkey` to contain the JSON request body.
// The operation will not return until the state has been updated.
// The result of the operation is `1` if a new entry was created and `0` if an existing entry was updated.
//
//     Consumes:
//     - application/json
//     Produces:
//     - text/plain
//     Schemes: http
//     Responses:
//       201: response201
//       204: response204
//       500: response500
//
func routeImplSet(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var mangledEntryKey string
	if subkey := ps.ByName("subkey"); subkey != "" {
		mangledEntryKey = nestedEntryKey(ps.ByName("key"), subkey)
	} else {
		mangledEntryKey = flatEntryKey(ps.ByName("key"))
	}

	if reply, err := store.HSet(stateKey(ps.ByName("type"), ps.ByName("id")), mangledEntryKey, ReadAll(r)); err != nil {
		http.Error(w, fmt.Sprintf("HSET failed: %v", err), http.StatusInternalServerError)
	} else if reply == 1 {
		if subkey := ps.ByName("subkey"); subkey != "" {
			w.Header().Set("Location", fmt.Sprintf("/kar/v1/actor/%v/%v/state/%v/%v", ps.ByName("type"), ps.ByName("id"), ps.ByName("key"), subkey))
		} else {
			w.Header().Set("Location", fmt.Sprintf("/kar/v1/actor/%v/%v/state/%v", ps.ByName("type"), ps.ByName("id"), ps.ByName("key")))

		}
		w.WriteHeader(http.StatusCreated) // New entry created
	} else {
		w.WriteHeader(http.StatusNoContent) // Existing entry updated
	}
}

// swagger:route GET /v1/actor/{actorType}/{actorId}/state/{key} state idActorStateGet
//
// state/key
//
// ### Get a single entry of an actor's state
//
// The `key` entry of the state of the actor instance indicated by `actorType` and `actorId`
// will be returned as the response body.
// If there is no entry for `key` a `404` response will be returned
// unless the boolean query parameter `nilOnAbsent` is set to `true`,
// in which case a `200` reponse with a `nil` response body will be returned.
//
//     Produces:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200StateGetResult
//       404: response404
//       500: response500
//

// swagger:route GET /v1/actor/{actorType}/{actorId}/state/{key}/{subkey} state idActorStateSubkeyGet
//
// state/key/subkey
//
// ### Get a single entry of an actor's state
//
// The `key/subkey` entry of the state of the actor instance indicated by `actorType` and `actorId`
// will be returned as the response body.
// If there is no entry for  `key/subkey` a `404` response will be returned
// unless the boolean query parameter `nilOnAbsent` is set to `true`,
// in which case a `200` reponse with a `nil` response body will be returned.
//
//     Produces:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200StateGetResult
//       404: response404
//       500: response500
//
func routeImplGet(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var mangledEntryKey string
	if subkey := ps.ByName("subkey"); subkey != "" {
		mangledEntryKey = nestedEntryKey(ps.ByName("key"), subkey)
	} else {
		mangledEntryKey = flatEntryKey(ps.ByName("key"))
	}

	if reply, err := store.HGet(stateKey(ps.ByName("type"), ps.ByName("id")), mangledEntryKey); err == store.ErrNil {
		if noa := r.FormValue("nilOnAbsent"); noa == "true" {
			fmt.Fprint(w, reply)
		} else {
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	} else if err != nil {
		http.Error(w, fmt.Sprintf("HGET failed: %v", err), http.StatusInternalServerError)
	} else {
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, reply)
	}
}

// swagger:route POST /v1/actor/{actorType}/{actorId}/state/{key} state idActorStateSubmapOps
//
// state/key
//
// ### Perform an operation on the actor map `key`
//
// The operation indicated by the `op` field of the request body will be performed on the `key` map
// of the actor instance indicated by `actorType` and `actorId`. The result of the
// operation will be returned as the response body.
// If there are no `key/subkey` entries in the actor instance, the operation
// will be interpreted as being applied to an empty map.
//
// The valid values for `op` are:
// <ul>
// <li>clear: remove all entires in the key actor map</li>
// <li>get: get the entire key actor map</li>
// <li>keys: return a list of subkeys that are defined in the key actor map</li>
// <li>size: return the number of entries the key actor map</li>
// </ul>
//
//     Consumes:
//     - application/json
//     Produces:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200StateSubmapOps
//       404: response404
//       500: response500
//
func routeImplSubmapOps(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var op submapOp
	if err := json.Unmarshal([]byte(ReadAll(r)), &op); err != nil {
		http.Error(w, "Request body was not a MapOp", http.StatusBadRequest)
		return
	}

	stateKey := stateKey(ps.ByName("type"), ps.ByName("id"))
	mapName := ps.ByName("key")

	var response interface{}
	switch op.Op {
	case "clear":
		mapKeys := []string{}
		err := subMapScan(stateKey, mapName, func(key string, value string) error {
			mapKeys = append(mapKeys, key)
			return nil
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("submapOps:clear: subMapScan failed: %v", err), http.StatusInternalServerError)
			return
		}
		numCleared, err := store.HDelMultiple(stateKey, mapKeys)
		if err != nil {
			http.Error(w, fmt.Sprintf("submapOps: HDEL failed  %v", err), http.StatusInternalServerError)
			return
		}
		response = numCleared

	case "get":
		m := map[string]interface{}{}
		subkeyPrefix := nestedEntryKeyPrefix(mapName)
		err := subMapScan(stateKey, mapName, func(key string, value string) error {
			if value != "" {
				userSubkey := strings.TrimPrefix(key, subkeyPrefix)
				var userValue interface{}
				if json.Unmarshal([]byte(value), &userValue) != nil {
					return fmt.Errorf("Failed to deserialize submap value: %v", value)
				}
				m[userSubkey] = userValue
			}
			return nil
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("submapOps:get: subMapScan failed: %v", err), http.StatusInternalServerError)
			return
		}
		response = m

	case "keys":
		userKeys := []string{}
		subkeyPrefix := nestedEntryKeyPrefix(mapName)
		err := subMapScan(stateKey, mapName, func(key string, value string) error {
			userKeys = append(userKeys, strings.TrimPrefix(key, subkeyPrefix))
			return nil
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("submapOps:keys: subMapScan failed: %v", err), http.StatusInternalServerError)
			return
		}
		response = userKeys

	case "size":
		size := 0
		err := subMapScan(stateKey, mapName, func(key string, value string) error {
			size = size + 1
			return nil
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("submapOps:size: subMapScan failed: %v", err), http.StatusInternalServerError)
			return
		}
		response = size

	default:
		http.Error(w, fmt.Sprintf("Unsupported map operation %v", op.Op), http.StatusBadRequest)
		return
	}

	buf, err := json.Marshal(response)
	if err != nil {
		http.Error(w, fmt.Sprintf("mapOps: error marshalling response %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	fmt.Fprint(w, string(buf))
}

func subMapScan(stateKey string, mapName string, op func(key string, value string) error) error {
	cursor := 0
	subkeyPrefix := nestedEntryKeyPrefix(mapName) + "*"
	for {
		newCursor, result, err := store.HScan(stateKey, cursor, subkeyPrefix)
		if err != nil {
			return err
		}
		cursor = newCursor
		for curIndex := 0; curIndex < len(result); curIndex += 2 {
			if innerErr := op(result[curIndex], result[curIndex+1]); innerErr != nil {
				return innerErr
			}
		}
		if cursor == 0 {
			break
		}
	}

	return nil
}

// swagger:route DELETE /v1/actor/{actorType}/{actorId}/state/{key} state idActorStateDelete
//
// state/key
//
// ### Remove a single entry in an actor's state
//
// The state of the actor instance indicated by `actorType` and `actorId`
// will be updated by removing the entry for `key`.
// The operation will not return until the state has been updated.
// The result of the operation is `1` if an entry was actually removed and
// `0` if there was no entry for `key`.
//
//     Schemes: http
//     Produces:
//     - text/plain
//     Responses:
//       200: response200StateDeleteResult
//       500: response500
//

// swagger:route DELETE /v1/actor/{actorType}/{actorId}/state/{key}/{subkey} state idActorStateSubkeyDelete
//
// state/key/subkey
//
// ### Remove a single entry in an actor's state
//
// The state of the actor instance indicated by `actorType` and `actorId`, and `key`
// will be updated by removing the entry for `key/subkey`.
// The operation will not return until the state has been updated.
// The result of the operation is `1` if an entry was actually removed and
// `0` if there was no entry for `key`.
//
//     Schemes: http
//     Produces:
//     - text/plain
//     Responses:
//       200: response200StateDeleteResult
//       500: response500
//
func routeImplDel(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var mangledEntryKey string
	if subkey := ps.ByName("subkey"); subkey != "" {
		mangledEntryKey = nestedEntryKey(ps.ByName("key"), subkey)
	} else {
		mangledEntryKey = flatEntryKey(ps.ByName("key"))
	}
	if reply, err := store.HDel(stateKey(ps.ByName("type"), ps.ByName("id")), mangledEntryKey); err != nil {
		http.Error(w, fmt.Sprintf("HDEL failed: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, reply)
	}
}

// swagger:route GET /v1/actor/{actorType}/{actorId}/state state idActorStateGetAll
//
// state
//
// ### Get an actor's state
//
// The state of the actor instance indicated by `actorType` and `actorId`
// will be returned as the response body.
//
//     Produces:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200StateGetAllResult
//       500: response500
//
func routeImplGetAll(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	state, err := actorGetAllState(ps.ByName("type"), ps.ByName("id"))
	if err != nil {
		http.Error(w, fmt.Sprintf("actorGetAllState failed: %v", err), http.StatusInternalServerError)
	} else {
		b, _ := json.Marshal(state)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, string(b))
	}
}

func actorGetAllState(actorType string, actorID string) (map[string]interface{}, error) {
	reply, err := store.HGetAll(stateKey(actorType, actorID))
	if err != nil {
		return nil, err
	}
	// reply has type map[string]string
	// we unmarshal the values then marshal the map
	m := map[string]interface{}{}
	for i, s := range reply {
		var v interface{}
		json.Unmarshal([]byte(s), &v)
		splitKeys := strings.SplitN(i, config.Separator, 2)
		key := splitKeys[0]
		subkey := splitKeys[1]
		if subkey == config.Separator {
			m[key] = v
		} else {
			if m[key] == nil {
				m[key] = map[string]interface{}{}
			}
			(m[key].(map[string]interface{}))[subkey] = v
		}
	}
	return m, nil
}

// swagger:route POST /v1/actor/{actorType}/{actorId}/state state idActorStateUpdate
//
// state
//
// ### Perform a multi-element update operation on the actor's state
//
// The state updates contained in the request body will be performed on the
// actor instance indicated by `actorType` and `actorId`.
// All removal operations will be performed first, then all update
// operations will be performed.
// The result of the operation will contain the number of state elements
// removed and updated.
//
//     Consumes:
//     - application/json
//     Produces:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200StateUpdate
//       404: response404
//       500: response500
//
func routeImplStateUpdate(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var op stateUpdateOp
	var err error
	if err = json.Unmarshal([]byte(ReadAll(r)), &op); err != nil {
		http.Error(w, "Request body was not a StateUpdateOp", http.StatusBadRequest)
		return
	}

	stateKey := stateKey(ps.ByName("type"), ps.ByName("id"))
	var response interface{}

	// First build up the keys to remove from the actor state
	toClear := []string{}
	for _, key := range op.Removals {
		toClear = append(toClear, flatEntryKey(key))
	}
	for mapName, removals := range op.SubmapRemovals {
		for _, subkey := range removals {
			toClear = append(toClear, nestedEntryKey(mapName, subkey))
		}
	}

	// Second construct the updates to apply to the actor state
	toUpdate := map[string]string{}
	for k, v := range op.Updates {
		s, err := json.Marshal(v)
		if err != nil {
			http.Error(w, fmt.Sprintf("StateUpdate: malformed update %v[%v] = %v. Error was %v", stateKey, k, v, err), http.StatusBadRequest)
			return
		}
		toUpdate[flatEntryKey(k)] = string(s)
	}
	for mapName, updates := range op.SubmapUpdates {
		for k, v := range updates {
			s, err := json.Marshal(v)
			if err != nil {
				http.Error(w, fmt.Sprintf("StateUpdate: malformed update %v[%v][%v] = %v. Error was %v", stateKey, mapName, k, v, err), http.StatusBadRequest)
				return
			}
			toUpdate[nestedEntryKey(mapName, k)] = string(s)
		}
	}

	// Third, apply the removals and then the updates.
	numCleared := 0
	numAdded := 0
	if len(toClear) > 0 {
		numCleared, err = store.HDelMultiple(stateKey, toClear)
		if err != nil {
			http.Error(w, fmt.Sprintf("StateUpate: HDEL failed  %v", err), http.StatusInternalServerError)
			return
		}
	}
	if len(toUpdate) > 0 {
		numAdded, err = store.HSetMultiple(stateKey, toUpdate)
		if err != nil {
			http.Error(w, fmt.Sprintf("StateUpate: HSET failed  %v", err), http.StatusInternalServerError)
			return
		}
	}

	response = response200StateUpdateOp{Removed: numCleared, Added: numAdded}
	buf, err := json.Marshal(response)
	if err != nil {
		http.Error(w, fmt.Sprintf("StateUpdate: error marshalling response %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	fmt.Fprint(w, string(buf))
}

// swagger:route DELETE /v1/actor/{actorType}/{actorId}/state state idActorStateDeleteAll
//
// state
//
// ### Remove an actor's state
//
// The state of the actor instance indicated by `actorType` and `actorId`
// will be deleted.
//
//     Schemes: http
//     Responses:
//       200: response200StateDeleteResult
//       404: response404
//       500: response500
//
func routeImplDelAll(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if reply, err := store.Del(stateKey(ps.ByName("type"), ps.ByName("id"))); err == store.ErrNil {
		http.Error(w, "Not Found", http.StatusNotFound)
	} else if err != nil {
		http.Error(w, fmt.Sprintf("DEL failed: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, reply)
	}
}
