package runtime

import (
	"encoding/json"
	//"context"
	"net/http"
	"fmt"
	"time"
	"strconv"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/IBM/kar/core/pkg/rpc"
)

func mapget(m map[string]string, key string, def string) string{
	val, ok := m[key]
	if !ok { return def }
	return val
}

func routeImplSetBreakpoint(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Println("hello")
	// golang can be very annoying sometimes
	// forward declaration of vars to make goto work
	var breakpointId string
	var msg map[string]string
	var msgBytes []byte
	var doCall func(string) error
	var node string
	var ok bool
	var actorType string
	var path string
	var retMap map[string]string
	var retBytes []byte
	var retval string

	body := ReadAll(r)
	var bodyJson map[string]string
	err := json.Unmarshal([]byte(body), &bodyJson)
	if err != nil { goto errorEncountered }

	breakpointId = "bk-"+uuid.New().String()
	retMap = map[string]string {
		"breakpointId": breakpointId,
	}
	retBytes, err = json.Marshal(retMap)
	if err != nil { goto errorEncountered }
	retval = string(retBytes)

	path, ok = bodyJson["path"]
	if !ok {
		err = fmt.Errorf("Path is not an optional argument")
		goto errorEncountered
	}

	actorType, ok = bodyJson["actorType"]
	if !ok {
		err = fmt.Errorf("Actor type is not an optional argument")
		goto errorEncountered
	}

	msg = map[string]string {
		"command": "setBreakpoint",
		"breakpointId": breakpointId,
		"breakpointType": mapget(bodyJson, "breakpointType", "global"),
		"actorId": mapget(bodyJson, "actorId", ""),
		"actorType": actorType,
		"path": path,
		"isCaller": mapget(bodyJson, "isCaller", "caller"),
		"isRequest": mapget(bodyJson, "isRequest", "request"),
	}
	msgBytes, err = json.Marshal(msg)
	if err != nil { goto errorEncountered }

	doCall = func(sidecar string) error {
		fmt.Println("doing call to "+sidecar)
		bytes, err := rpc.Call(ctx, rpc.Destination{Target: rpc.Node{ID: sidecar}, Method: sidecarEndpoint}, time.Time{}, "", msgBytes)
		if err != nil { return err }
		var reply Reply
		err = json.Unmarshal(bytes, &reply)
		if err != nil { return err }
		if reply.StatusCode != 200 {
			err = fmt.Errorf("Status code of reply not OK: %v", reply.StatusCode)
			return err
		}
		return nil
	}

	node, ok = bodyJson["node"]
	if !ok || node == "" {
		sidecars, _ := rpc.GetNodeIDs()
		for _, sidecar := range sidecars {
			if /*sidecar != rpc.GetNodeID()*/ true {
				// TODO: error handling
				// right now, errors might leave
				// nodes in inconsistent state w.r.t.
				// breakpoints

				// TODO: parallelize rpcs

				err = doCall(sidecar)
				if err != nil { goto errorEncountered }
			}
		}
	} else {
		err = doCall(node)
		if err != nil { goto errorEncountered }
	}
	// assume no error at this point
	w.Header().Add("Content-Type", "application/kar+json")
	w.WriteHeader(200)
	fmt.Fprint(w, string(retval))
	return

errorEncountered:
	http.Error(w, fmt.Sprintf("failed to set breakpoint: %v", err), http.StatusInternalServerError)
}

func routeImplUnsetBreakpoint(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var err error
	errorFunc := func() {
		http.Error(w, fmt.Sprintf("failed to unset breakpoint: %v", err), http.StatusInternalServerError)
	}

	body := ReadAll(r)
	var bodyJson map[string]string
	err = json.Unmarshal([]byte(body), &bodyJson)
	if err != nil { errorFunc(); return }

	breakpointId, ok := bodyJson["breakpointId"]
	if !ok {
		err = fmt.Errorf("argument breakpointId not given")
		errorFunc(); return
	}

	var msg = map[string]string {
		"command": "unsetBreakpoint",
		"breakpointId": breakpointId,
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil { errorFunc(); return }

	doCall := func(sidecar string) error {
		bytes, err := rpc.Call(ctx, rpc.Destination{Target: rpc.Node{ID: sidecar}, Method: sidecarEndpoint}, time.Time{}, "", msgBytes)
		if err != nil { return err }
		var reply Reply
		err = json.Unmarshal(bytes, &reply)
		if err != nil { return err }
		if reply.StatusCode != 200 {
			err = fmt.Errorf("Status code of reply not OK: %v", reply.StatusCode)
			return err
		}
		return nil
	}

	node, ok := bodyJson["node"]
	if !ok || node == "" {
		sidecars, _ := rpc.GetNodeIDs()
		for _, sidecar := range sidecars {
			if sidecar != rpc.GetNodeID() {
				// TODO: error handling
				// right now, errors might leave
				// nodes in inconsistent state w.r.t.
				// breakpoints

				// TODO: parallelize rpcs

				err = doCall(sidecar)
				if err != nil { errorFunc(); return }
			}
		}
	} else {
		err = doCall(node)
		if err != nil { errorFunc(); return }
	}
	// assume no error at this point
	w.WriteHeader(200)
}

func routeImplRegisterDebugger(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	isDebugger = true

	body := ReadAll(r)
	var bodyJson map[string]string
	err := json.Unmarshal([]byte(body), &bodyJson)
	if err == nil {
		myDebuggerAppHost, ok := bodyJson["host"]
		if ok {
			debuggerAppHost = myDebuggerAppHost
		}

		myDebuggerAppPortStr, ok := bodyJson["port"]
		if ok {
			myDebuggerAppPort, err := strconv.Atoi(myDebuggerAppPortStr)
			if err == nil { debuggerAppPort = myDebuggerAppPort }
		}
	}

	// register debugger with all sidecars
	var msg = map[string]string {
		"command": "registerDebugger",
		"debuggerId": rpc.GetNodeID(),
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil { return }

	doTell := func(sidecar string){
		rpc.Tell(ctx, rpc.Destination{Target: rpc.Node{ID: sidecar}, Method: sidecarEndpoint}, time.Time{}, msgBytes)
	}

	sidecars, _ := rpc.GetNodeIDs()
	for _, sidecar := range sidecars {
		if true /*sidecar != rpc.GetNodeID()*/ {
			doTell(sidecar)
		}
	}

	w.WriteHeader(200)
}

func routeImplUnregisterDebugger(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	isDebugger = false

	// register debugger with all sidecars
	var msg = map[string]string {
		"command": "unregisterDebugger",
		"debuggerId": rpc.GetNodeID(),
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil { return }

	doTell := func(sidecar string){
		rpc.Tell(ctx, rpc.Destination{Target: rpc.Node{ID: sidecar}, Method: sidecarEndpoint}, time.Time{}, msgBytes)
	}

	sidecars, _ := rpc.GetNodeIDs()
	for _, sidecar := range sidecars {
		if sidecar != rpc.GetNodeID() {
			doTell(sidecar)
		}
	}

	w.WriteHeader(200)
}


func routeImplPause(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var err error
	errorFunc := func() {
		http.Error(w, fmt.Sprintf("failed to pause node: %v", err), http.StatusInternalServerError)
	}

	body := ReadAll(r)
	var bodyJson map[string]string
	err = json.Unmarshal([]byte(body), &bodyJson)
	if err != nil { errorFunc(); return }

	_, actorTypeOk := bodyJson["actorType"]
	_, actorIdOk := bodyJson["actorId"]

	if !actorTypeOk && actorIdOk {
		err = fmt.Errorf("if actorId is set, then actorType must be set")
		errorFunc(); return
	}

	var msg = map[string]string {
		"command": "pause",
		"actorType": mapget(bodyJson, "actorType", ""),
		"actorId": mapget(bodyJson, "actorId", ""),
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil { errorFunc(); return }

	doCall := func(sidecar string) error {
		bytes, err := rpc.Call(ctx, rpc.Destination{Target: rpc.Node{ID: sidecar}, Method: sidecarEndpoint}, time.Time{}, "", msgBytes)
		if err != nil { return err }
		var reply Reply
		err = json.Unmarshal(bytes, &reply)
		if err != nil { return err }
		if reply.StatusCode != 200 {
			err = fmt.Errorf("Status code of reply not OK: %v", reply.StatusCode)
			return err
		}
		return nil
	}

	node, ok := bodyJson["node"]
	if !ok || node == "" {
		sidecars, _ := rpc.GetNodeIDs()
		for _, sidecar := range sidecars {
			if sidecar != rpc.GetNodeID() {
				// TODO: error handling
				// right now, errors might leave
				// nodes in inconsistent state w.r.t.
				// breakpoints

				// TODO: parallelize rpcs

				err = doCall(sidecar)
				if err != nil { errorFunc(); return }
			}
		}
	} else {
		err = doCall(node)
		if err != nil { errorFunc(); return }
	}
	// assume no error at this point
	w.WriteHeader(200)
}

func routeImplUnpause(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var err error
	errorFunc := func() {
		http.Error(w, fmt.Sprintf("failed to unpause node: %v", err), http.StatusInternalServerError)
	}

	body := ReadAll(r)
	var bodyJson map[string]string
	err = json.Unmarshal([]byte(body), &bodyJson)
	if err != nil { errorFunc(); return }

	_, actorTypeOk := bodyJson["actorType"]
	_, actorIdOk := bodyJson["actorId"]

	if !actorTypeOk && actorIdOk {
		err = fmt.Errorf("if actorId is set, then actorType must be set")
		errorFunc(); return
	}

	var msg = map[string]string {
		"command": "unpause",
		"actorType": mapget(bodyJson, "actorType", ""),
		"actorId": mapget(bodyJson, "actorId", ""),
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil { errorFunc(); return }

	doCall := func(sidecar string) error {
		bytes, err := rpc.Call(ctx, rpc.Destination{Target: rpc.Node{ID: sidecar}, Method: sidecarEndpoint}, time.Time{}, "", msgBytes)
		if err != nil { return err }
		var reply Reply
		err = json.Unmarshal(bytes, &reply)
		if err != nil { return err }
		if reply.StatusCode != 200 {
			err = fmt.Errorf("Status code of reply not OK: %v", reply.StatusCode)
			return err
		}
		return nil
	}

	node, ok := bodyJson["node"]
	if !ok || node == "" {
		sidecars, _ := rpc.GetNodeIDs()
		for _, sidecar := range sidecars {
			if sidecar != rpc.GetNodeID() {
				// TODO: error handling
				// right now, errors might leave
				// nodes in inconsistent state w.r.t.
				// breakpoints

				// TODO: parallelize rpcs

				err = doCall(sidecar)
				if err != nil { errorFunc(); return }
			}
		}
	} else {
		err = doCall(node)
		if err != nil { errorFunc(); return }
	}
	// assume no error at this point
	w.WriteHeader(200)
}
