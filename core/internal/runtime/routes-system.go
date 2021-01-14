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
 * KAR REST API related to system-level operations.
 */

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.ibm.com/solsa/kar.git/core/internal/pubsub"
	"github.ibm.com/solsa/kar.git/core/pkg/logger"
)

// swagger:route POST /v1/system/shutdown system idSystemShutdown
//
// shutdown
//
// ### Shutdown a single KAR runtime
//
// Initiate an orderly shutdown of the target KAR runtime process.
//
//     Schemes: http
//     Responses:
//       200: response200
//
func routeImplShutdown(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprint(w, "OK")
	logger.Info("Invoking cancel() in response to shutdown request")
	cancel()
}

// swagger:route GET /v1/system/health system isSystemHealth
//
// health
//
// ### Health-check endpoint
//
// Returns a `200` response to indicate that the KAR runtime processes is healthy.
//
//     Schemes: http
//     Responses:
//       200: response200
//
func routeImplHealth(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprint(w, "OK")
}

// post handles a direct http request from a peer sidecar
// TODO swagger
func routeImplPost(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	value, _ := ioutil.ReadAll(r.Body)
	m := pubsub.Message{Value: value}
	process(m)
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprint(w, "OK")
}

// Returns information about a specified component, controlled by the call path
// Options are given by the cases
// Format type (text/plain vs application/json) is controlled by Accept header in call
func routeImplGetInformation(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	format := "text/plain"
	if r.Header.Get("Accept") == "application/json" {
		format = "application/json"
	}
	component := ps.ByName("component")
	var data string
	var err error
	switch component {
	case "id":
		data, err = pubsub.GetSidecarID(format)
	case "sidecars", "Sidecars":
		data, err = pubsub.GetSidecars(format)
	case "actors", "Actors":
		if actorMap, err := getAllActiveActors(ctx, ""); err == nil {
			data, err = formatActorInstanceMap(actorMap, format)
		}
	case "sidecar_actors":
		data, err = formatActorInstanceMap(getMyActiveActors(""), format)
	default:
		http.Error(w, fmt.Sprintf("Invalid information query: %v", component), http.StatusBadRequest)
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to acquire %v information: %v", component, err), http.StatusInternalServerError)
	} else {
		w.Header().Add("Content-Type", format)
		fmt.Fprint(w, data)
	}
}
