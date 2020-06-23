package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
		logger.Error("internal error: %v", err)
		exitCode = 1
		return
	}
	reply, err := CallActor(ctx, actor, path, string(payload), "application/kar+json", "application/json", "")
	if err != nil {
		logger.Error("internal error: %v", err)
		exitCode = 1
		return
	}
	var r map[string]interface{}
	err = json.Unmarshal([]byte(reply.Payload), &r)
	if err != nil {
		logger.Error("internal error: %v", err)
		exitCode = 1
		return
	}
	if r["error"] != nil {
		dump("[STDERR] ", strings.NewReader(fmt.Sprintf("%v", r["message"])))
		dump("[STDERR] ", strings.NewReader(fmt.Sprintf("%v", r["stack"])))
	} else {
		dump("[STDOUT] ", strings.NewReader(fmt.Sprintf("%v", r["value"])))
	}
	return
}
