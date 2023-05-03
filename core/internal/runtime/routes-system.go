//
// Copyright IBM Corporation 2020,2023
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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/IBM/kar/core/pkg/logger"
	"github.com/IBM/kar/core/pkg/rpc"
	"github.com/julienschmidt/httprouter"
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

// swagger:route GET /v1/system/health system idSystemHealth
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

// swagger:route GET /v1/system/information/{component} system idSystemInfo
//
// information
//
// ### System information
//
// Returns information about a specified component, controlled by the call path
//
//     Schemes: http
//     Produces:
//     - text/plain
//     - application/json
//     Responses:
//       200: response200SystemInfoResult
//
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
		nodeID := rpc.GetNodeID()
		if format == "json" || format == "application/json" {
			data = fmt.Sprintf("{\"id\":\"%s\"}", nodeID)
		} else {
			data = nodeID + "\n"
		}
		err = nil
	case "sidecars", "Sidecars":
		karTopology := make(map[string]sidecarData)
		topology, _ := rpc.GetTopology()
		ports, _ := rpc.GetPorts()
		for node, services := range topology {
			karTopology[node] = sidecarData{Port: ports[node], Services: []string{services[0]}, Actors: services[1:]}
		}

		if format == "json" || format == "application/json" {
			m, err := json.Marshal(karTopology)
			if err == nil {
				data = string(m)
			}
		} else {
			var str strings.Builder
			fmt.Fprint(&str, "\nSidecar\n : Actors\n : Services")
			for sidecar, sidecarInfo := range karTopology {
				fmt.Fprintf(&str, "\n%v:%v\n : %v\n : %v", sidecar, sidecarInfo.Port, sidecarInfo.Actors, sidecarInfo.Services)
			}
			data, err = str.String(), nil
		}
	case "actors", "Actors":
		if actorMap, err := getAllActiveActors(ctx, ""); err == nil {
			data, err = formatActorInstanceMap(actorMap, format)
		}
	case "sidecar_actors":
		data, err = formatActorInstanceMap(rpc.GetLocalActivatedSessions(ctx, ""), format)
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

type sidecarData struct {
	Port     int32    `json:"port"`
	Actors   []string `json:"actors"`
	Services []string `json:"services"`
	//Host string `json:"host"`
}
