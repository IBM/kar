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

	"github.com/julienschmidt/httprouter"
	"github.ibm.com/solsa/kar.git/core/internal/config"
	"github.ibm.com/solsa/kar.git/core/internal/store"
	"github.ibm.com/solsa/kar.git/core/pkg/logger"
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
// The result of the operation is `1` if a new entry was created and `0` if an existing entry was updated.
//
//     Consumes:
//     - application/json
//     Produces:
//     - text/plain
//     Schemes: http
//     Responses:
//       200: response200StateSetResult
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
//       200: response200StateSetResult
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
	} else {
		fmt.Fprint(w, reply)
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

// swagger:route POST /v1/actor/{actorType}/{actorId}/state/{key} state idActorStateMapOps
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
// <li>update: update the key actor map to contain all the subkey to value mappings contained in updates</li>
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
		}
		response = numCleared

	case "get":
		mapKeys, err := getSubMapKeys(stateKey, mapName)
		if err != nil {
			http.Error(w, fmt.Sprintf("HKEYS failed: %v", err), http.StatusInternalServerError)
			return
		}

		// Construct the response map by looking up each subkey
		subkeyPrefix := nestedEntryKeyPrefix(mapName)
		m := map[string]interface{}{}
		for i := range mapKeys {
			if vstr, err := store.HGet(stateKey, mapKeys[i]); err == store.ErrNil {
				// Map contains nil for this key; elide the entry
			} else if err != nil {
				http.Error(w, fmt.Sprintf("HKEY failed: %v", err), http.StatusInternalServerError)
				return
			} else {
				userSubkey := strings.TrimPrefix(mapKeys[i], subkeyPrefix)
				var userValue interface{}
				if json.Unmarshal([]byte(vstr), &userValue) != nil {
					http.Error(w, fmt.Sprintf("Failed to deserialize result of HGET: %v", err), http.StatusInternalServerError)
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
	if reply, err := store.HGetAll(stateKey(ps.ByName("type"), ps.ByName("id"))); err != nil {
		http.Error(w, fmt.Sprintf("HGETALL failed: %v", err), http.StatusInternalServerError)
	} else {
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
		b, _ := json.Marshal(m)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, string(b))
	}
}

// swagger:route POST /v1/actor/{actorType}/{actorId}/state state idActorStateSetMultiple
//
// state
//
// ### Update multiple entries of an actor's state
//
// The state of the actor instance indicated by `actorType` and `actorId`
// will be updated by atomically updated by storing all key-value pairs
// in the request body.
// The operation will not return until the state has been updated.
// The result of the operation is the number of new entires that were created.
//
//     Consumes:
//     - application/json
//     Produces:
//     - text/plain
//     Schemes: http
//     Responses:
//       200: response200StateSetMultipleResult
//       400: response400
//       500: response500
//
func routeImplSetMultiple(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var updates map[string]interface{}
	if err := json.Unmarshal([]byte(ReadAll(r)), &updates); err != nil {
		http.Error(w, "Request body was not a map[string, interface{}]", http.StatusBadRequest)
		return
	}
	stateKey := stateKey(ps.ByName("type"), ps.ByName("id"))
	if reply, err := actorSetMultiple(stateKey, config.Separator, updates); err != nil {
		http.Error(w, fmt.Sprintf("setMultiple failed: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, reply)
	}
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
