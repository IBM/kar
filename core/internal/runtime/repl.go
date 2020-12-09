package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.ibm.com/solsa/kar.git/core/internal/config"
	"github.ibm.com/solsa/kar.git/core/internal/pubsub"
	"github.ibm.com/solsa/kar.git/core/pkg/logger"
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
	reply, err := CallActor(ctx, actor, path, string(payload), "", false)
	if err != nil {
		logger.Error("error invoking the actor: %v", err)
		exitCode = 1
		return
	}
	if reply.StatusCode == http.StatusNoContent {
		log.Printf("[STDOUT] Method completed normally with void result.")
	} else if reply.StatusCode == http.StatusOK {
		if strings.HasPrefix(reply.ContentType, "application/kar+json") {
			var result actorCallResult
			if err := json.Unmarshal([]byte(reply.Payload), &result); err != nil {
				log.Printf("[STDERR] Internal error: malformed method result: %v", err)
			} else {
				if result.Error {
					log.Printf("[STDERR] Exception raised: %s", result.Message)
					log.Printf("[STDERR] Stacktrace: %v", result.Stack)
				} else {
					log.Printf("[STDOUT] Method result: %v", result.Value)
				}
			}
		} else {
			log.Printf("[STDOUT] %v", reply.Payload)
		}
	} else {
		log.Printf("[STDERR] HTTP status: %v", reply.StatusCode)
		log.Printf("[STDERR] %v", reply.Payload)
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

	reply, err := CallService(ctx, service, path, body, header, method, false)
	if err != nil {
		logger.Error("error invoking the service: %v", err)
		exitCode = 1
		return
	}
	if reply.StatusCode != http.StatusOK {
		log.Printf("[STDERR] HTTP status: %v", reply.StatusCode)
		log.Printf("[STDERR] %v", reply.Payload)
	} else {
		log.Printf("[STDOUT] %v", reply.Payload)
	}
	return
}

func getInformation(ctx context.Context, args []string) (exitCode int) {
	option := strings.ToLower(config.GetSystemComponent)
	var str string
	var err error
	switch option {
	case "sidecar", "sidecars":
		str, err = pubsub.GetSidecars(config.GetOutputStyle)
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
				if actorMap, err := pubsub.GetAllActorInstances(config.GetActorType); err == nil {
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
	log.Printf("%s", str)
	return
}
