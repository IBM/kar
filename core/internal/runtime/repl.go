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
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/IBM/kar/core/internal/config"
	"github.com/IBM/kar/core/pkg/logger"
	"github.com/IBM/kar/core/pkg/rpc"
)

// invokeActorMethod an actor method
func invokeActorMethod(ctx context.Context, args []string) (exitCode int) {
	actor := Actor{Type: args[0], ID: args[1]}
	path := "/" + args[2]
	params := make([]interface{}, len(args[3:]))
	for i, a := range args[3:] {
		if json.Unmarshal([]byte(a), &params[i]) != nil {
			params[i] = args[3+i]
			logger.Warning("assuming argument %v is a string", params[i])
		}
	}
	payload, err := json.Marshal(params)
	if err != nil {
		logger.Error("error serializing the payload: %v", err)
		exitCode = 1
		return
	}
	reply, err := CallActor(ctx, actor, path, string(payload), "", "")
	if err != nil {
		logger.Error("error invoking the actor: %v", err)
		exitCode = 1
		return
	}
	if reply.StatusCode == http.StatusNoContent {
		fmt.Println("Method completed normally with void result.")
	} else if reply.StatusCode == http.StatusOK {
		if strings.HasPrefix(reply.ContentType, "application/kar+json") {
			var result actorCallResult
			if err := json.Unmarshal([]byte(reply.Payload), &result); err != nil {
				fmt.Fprintf(os.Stderr, "[STDERR] Internal error: malformed method result: %v\n", err)
			} else {
				if result.Error {
					fmt.Fprintf(os.Stderr, "[STDERR] Exception raised: %s\n", result.Message)
					fmt.Fprintf(os.Stderr, "[STDERR] Stacktrace: %v\n", result.Stack)
				} else {
					fmt.Printf("Method result: %v\n", result.Value)
				}
			}
		} else {
			fmt.Println(reply.Payload)
		}
	} else {
		fmt.Fprintf(os.Stderr, "[STDERR] HTTP status: %v\n", reply.StatusCode)
		fmt.Fprintf(os.Stderr, "[STDERR] %v\n", reply.Payload)
	}
	return
}

// invokeServiceEndpoint makes a request to a service endpoint
func invokeServiceEndpoint(ctx context.Context, args []string) (exitCode int) {
	method := strings.ToUpper(args[0])
	service := args[1]
	path := "/" + args[2]
	var header, body string
	if len(args) > 3 {
		body = args[3]
		header = fmt.Sprintf("{\"Content-Type\": [\"%v\"]}", config.RestBodyContentType)
	} else {
		header = ""
		body = ""
	}

	reply, err := CallService(ctx, service, path, body, header, method)
	if err != nil {
		logger.Error("error invoking the service: %v", err)
		exitCode = 1
		return
	}
	if reply.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "[STDERR] HTTP status: %v\n", reply.StatusCode)
		fmt.Fprintf(os.Stderr, "[STDERR] %v\n", reply.Payload)
	} else {
		fmt.Println(reply.Payload)
	}
	return
}



type addrTuple_t struct {
	Host string `json:"host"`
	Port int `json:"port"`
}

func getInformation(ctx context.Context, args []string) (exitCode int) {
	option := strings.ToLower(config.GetSystemComponent)
	var str string
	var err error
	switch option {
	case "sidecar", "sidecars":
		doCall := func(sidecar string) (addrTuple_t, error) {
			bytes, err := rpc.Call(ctx, rpc.Destination{Target: rpc.Node{ID: sidecar}, Method: sidecarEndpoint}, time.Time{}, "", []byte("{\"command\": \"getRuntimeAddr\"}") )
			if err != nil { return addrTuple_t{}, err }
			var reply Reply
			err = json.Unmarshal(bytes, &reply)
			if err != nil { return addrTuple_t{}, err }
			if reply.StatusCode != 200 {
				err = fmt.Errorf("Status code of reply not OK: %v", reply.StatusCode)
				return addrTuple_t{}, err
			}
			var addr = addrTuple_t {}
			err = json.Unmarshal([]byte(reply.Payload), &addr)
			if err != nil { return addrTuple_t{}, err }
			return addr, err
		}

		karTopology := make(map[string]sidecarData)
		topology, _ := rpc.GetTopology()
		for node, services := range topology {
			addr, err := doCall(node)
			if err != nil { addr = addrTuple_t {} }
			karTopology[node] = sidecarData{Services: []string{services[0]}, Actors: services[1:], Host: addr.Host, Port: addr.Port}
		}
		if config.GetOutputStyle == "json" || config.GetOutputStyle == "application/json" {
			m, err := json.Marshal(karTopology)
			if err == nil {
				str = string(m)
			}
		} else {
			var sb strings.Builder
			fmt.Fprint(&sb, "\nSidecar : host : port \n : Actors\n : Services")
			for sidecar, sidecarInfo := range karTopology {
				fmt.Fprintf(&sb, "\n%v", sidecar)
				//addr, err := doCall(sidecar)
				if err == nil {
					fmt.Fprintf(&sb, " : %v : %v",
						sidecarInfo.Host, sidecarInfo.Port)
				} else {
					fmt.Printf("error: %v", err)
				}
				if sidecar == rpc.GetNodeID() {
					fmt.Fprintf(&sb, " (this sidecar)")
				}
				fmt.Fprintf(&sb, "\n : %v\n : %v", sidecarInfo.Actors, sidecarInfo.Services)
			}
			str, err = sb.String(), nil
		}
	case "actor", "actors":
		if config.GetActorInstanceID != "" {
			if actorState, err := actorGetAllState(config.GetActorType, config.GetActorInstanceID); err == nil {
				if len(actorState) == 0 {
					str = fmt.Sprintf("Actor %v[%v] has no persisted state", config.GetActorType, config.GetActorInstanceID)
				} else {
					if bytes, err := json.MarshalIndent(actorState, "", "  "); err == nil {
						str = fmt.Sprintf("Persisted state of actor %v[%v] is:\n", config.GetActorType, config.GetActorInstanceID) + string(bytes)
					}
				}
			}
		} else {
			var prefix string
			if !config.GetResidentOnly {
				if config.GetActorType != "" {
					prefix = fmt.Sprintf("Known instances of actor type %v are:\n", config.GetActorType)
				} else {
					prefix = fmt.Sprintf("Listing all known actor instances:\n")
				}
				if actorMap, err := rpc.GetAllSessions(ctx, config.GetActorType); err == nil {
					str, err = formatActorInstanceMap(actorMap, config.GetOutputStyle)
				}
			} else {
				if config.GetActorType != "" {
					prefix = fmt.Sprintf("Memory-resident instances of actor type %v are:\n", config.GetActorType)
				} else {
					prefix = fmt.Sprintf("Listing all memory-resident actor instances:\n")
				}
				if actorMap, err := getAllActiveActors(ctx, config.GetActorType); err == nil {
					str, err = formatActorInstanceMap(actorMap, config.GetOutputStyle)
				}
			}
			if err == nil {
				str = prefix + str
			}
		}
	default:
		logger.Error("invalid argument <%v> to call Inform", option)
		exitCode = 1
		return
	}
	if err != nil {
		logger.Error("error in Get on %v: %v", option, err)
		exitCode = 1
		return
	}
	fmt.Println(str)
	return
}
