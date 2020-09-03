package runtime

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/internal/pubsub"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

// Invoke an actor method
func Invoke(ctx context.Context, args []string) (exitCode int) {
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
	reply, err := CallActor(ctx, actor, path, string(payload), "application/kar+json", "", "POST", "", false)
	if err != nil {
		logger.Error("error invoking the actor: %v", err)
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

// Perform information methods
func GetInformation(ctx context.Context, args []string) (exitCode int) {
	option := strings.ToLower(config.Get)
	switch option {
	case "sidecar", "sidecars":
		str, err := pubsub.GetSidecars(config.OutputStyle)
		if err != nil {
			logger.Error("error in Inform on %v: %v", option, err)
			exitCode = 1
			return
		}
		log.Printf(str)
	default:
		logger.Error("invalid argument <%v> to call Inform", option)
		exitCode = 1
		return
	}
	return
}
