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

	"github.com/IBM/kar.git/core/internal/config"
	"github.com/IBM/kar.git/core/internal/store"
	"github.com/IBM/kar.git/core/pkg/logger"
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
// <li>clearSome: remove entries in the key actor for the subkeys contained in removals</li>
// <li>get: get the entire key actor map</li>
// <li>keys: return a list of subkeys that are defined in the key actor map</li>
// <li>size: return the number of entries the key actor map</li>
// <li>update: update the key actor map to contain all the subkey to value mappings contained in updates</li>
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
	var op mapOp
	if err := json.Unmarshal([]byte(ReadAll(r)), &op); err != nil {
		http.Error(w, "Request body was not a MapOp", http.StatusBadRequest)
		return
	}

	stateKey := stateKey(ps.ByName("type"), ps.ByName("id"))
	mapName := ps.ByName("key")

	var response interface{}
	switch op.Op {
	case "clear":
		mapKeys, err := getSubMapKeys(stateKey, mapName)
		if err != nil {
			http.Error(w, fmt.Sprintf("mapOps: getSubMapKeys failed: %v", err), http.StatusInternalServerError)
			return
		}
		numCleared, err := store.HDelMultiple(stateKey, mapKeys)
		if err != nil {
			http.Error(w, fmt.Sprintf("mapOps: HDEL failed  %v", err), http.StatusInternalServerError)
			return
		}
		response = numCleared

	case "clearSome":
		mapKeys := []string{}
		for _, subkey := range op.Removals {
			mapKeys = append(mapKeys, nestedEntryKey(mapName, subkey))
		}
		numCleared, err := store.HDelMultiple(stateKey, mapKeys)
		if err != nil {
			http.Error(w, fmt.Sprintf("mapOps: HDEL failed  %v", err), http.StatusInternalServerError)
			return
		}
		response = numCleared

	case "get":
		mapKeys, err := getSubMapKeys(stateKey, mapName)
		if err != nil {
			http.Error(w, fmt.Sprintf("HKEYS failed: %v", err), http.StatusInternalServerError)
			return
		}
		mapVals, err := store.HMGet(stateKey, mapKeys)
		if err != nil {
			http.Error(w, fmt.Sprintf("HMGET failed: %v", err), http.StatusInternalServerError)
			return
		}

		// Construct the response map by splicing together mapKeys and mapVals
		subkeyPrefix := nestedEntryKeyPrefix(mapName)
		m := map[string]interface{}{}
		for i := range mapKeys {
			val := mapVals[i]
			if val != "" {
				userSubkey := strings.TrimPrefix(mapKeys[i], subkeyPrefix)
				var userValue interface{}
				if json.Unmarshal([]byte(val), &userValue) != nil {
					http.Error(w, fmt.Sprintf("Failed to deserialize value: %v", err), http.StatusInternalServerError)
					return
				}
				m[userSubkey] = userValue
			}
		}
		response = m

	case "keys":
		mapKeys, err := getSubMapKeys(stateKey, mapName)
		if err != nil {
			http.Error(w, fmt.Sprintf("mapOps: getSubMapKeys failed: %v", err), http.StatusInternalServerError)
			return
		}
		cleanedKeys := make([]string, 0, len(mapKeys))
		subkeyPrefix := nestedEntryKeyPrefix(mapName)
		for i := range mapKeys {
			cleanedKeys = append(cleanedKeys, strings.TrimPrefix(mapKeys[i], subkeyPrefix))
		}
		response = cleanedKeys

	case "size":
		mapKeys, err := getSubMapKeys(stateKey, mapName)
		if err != nil {
			http.Error(w, fmt.Sprintf("HKEYS failed: %v", err), http.StatusInternalServerError)
			return
		}
		response = len(mapKeys)

	case "update":
		numUpdated, err := actorSetMultiple(stateKey, mapName, op.Updates)
		if err != nil {
			http.Error(w, fmt.Sprintf("mapOps: setMultiple failed: %v", err), http.StatusInternalServerError)
			return
		}
		response = numUpdated

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

func getSubMapKeys(stateKey string, mapName string) ([]string, error) {
	// TODO: Future performance optimization.
	//       We should instead use HScan to incrementally accumulate the
	//       list of keys to avoid long latency operations on Redis.
	keys, err := store.HKeys(stateKey)
	if err != nil {
		return nil, err
	}
	subkeyPrefix := nestedEntryKeyPrefix(mapName)
	flatKey := flatEntryKey(mapName)
	mapKeys := []string{}
	for i := range keys {
		if keys[i] != flatKey && strings.HasPrefix(keys[i], subkeyPrefix) {
			mapKeys = append(mapKeys, keys[i])
		}
	}
	return mapKeys, nil
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

// swagger:route POST /v1/actor/{actorType}/{actorId}/state state idActorStateMapOps
//
// state
//
// ### Perform an operation on the actor's state
//
// The operation indicated by the `op` field of the request body will be performed on the
// actor instance indicated by `actorType` and `actorId`. The result of the
// operation will be returned as the response body.
// If there are no `key` entries in the actor instance, the operation
// will be interpreted as being applied to an empty map.
//
// The valid values for `op` are:
// <ul>
// <li>clearSome: remove entries in the key actor for the subkeys contained in removals</li>
// <li>update: atomically update the actor state to contain all the key-value pairs contained in updates</li>
// </ul>
//
//     Consumes:
//     - application/json
//     Produces:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200StateMapOps
//       404: response404
//       500: response500
//
func routeImplMapOps(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var op mapOp
	if err := json.Unmarshal([]byte(ReadAll(r)), &op); err != nil {
		http.Error(w, "Request body was not a MapOp", http.StatusBadRequest)
		return
	}

	stateKey := stateKey(ps.ByName("type"), ps.ByName("id"))
	var response interface{}

	switch op.Op {
	case "clearSome":
		mapKeys := []string{}
		for _, key := range op.Removals {
			mapKeys = append(mapKeys, flatEntryKey(key))
		}
		numCleared, err := store.HDelMultiple(stateKey, mapKeys)
		if err != nil {
			http.Error(w, fmt.Sprintf("mapOps: HDEL failed  %v", err), http.StatusInternalServerError)
			return
		}
		response = numCleared

	case "update":
		numSet, err := actorSetMultiple(stateKey, config.Separator, op.Updates)
		if err != nil {
			http.Error(w, fmt.Sprintf("setMultiple failed: %v", err), http.StatusInternalServerError)
		}
		response = numSet

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

func actorSetMultiple(stateKey string, key string, updates map[string]interface{}) (int, error) {
	m := map[string]string{}
	for i, v := range updates {
		s, err := json.Marshal(v)
		if err != nil {
			logger.Error("setMultiple: %v[%v] = %v failed due to %v", stateKey, i, v, err)
			return 0, err
		}
		if key == config.Separator {
			m[flatEntryKey(i)] = string(s)
		} else {
			m[nestedEntryKey(key, i)] = string(s)
		}
	}
	return store.HSetMultiple(stateKey, m)
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
