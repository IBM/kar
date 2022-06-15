package runtime

import (
	"encoding/json"
	//"context"
	"net/http"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/IBM/kar/core/pkg/rpc"
	"github.com/gorilla/websocket"
)

func mapget(m map[string]string, key string, def string) string{
	val, ok := m[key]
	if !ok { return def }
	return val
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func sendAll(bytes []byte, curConnId string) error {
	debugConnsLock.Lock()
	defer debugConnsLock.Unlock()

	for key, conn := range debugConns {
		err := conn.WriteMessage(websocket.TextMessage, bytes)
		if err != nil {
			closeConnNoLock(conn, key);
			if curConnId == key {
				return err
			}
		}
	}
	return nil
}
func closeConnNoLock(conn *websocket.Conn, key string){
	conn.Close()
	delete(debugConns, key)
	if len(debugConns) == 0 {
		// no connections left to this node
		// unregister this node as a debugger node
		// from all of the other sidecars
		unregisterDebuggerSidecar()
	}
}
func closeConn(conn *websocket.Conn, key string){
	conn.Close()
	debugConnsLock.Lock()
	delete(debugConns, key)
	if len(debugConns) == 0 {
		// no connections left to this node
		// unregister this node as a debugger node
		// from all of the other sidecars
		unregisterDebuggerSidecar()
	}
	debugConnsLock.Unlock()
}
func debugServe(debugConn *websocket.Conn, debuggerId string){
	sendErrorBytes := func(err error, cmd string) error {
		errorMap := map[string]string {
			"command": cmd,
			"error": fmt.Sprintf("%v", err),
		}
		errorBytes, _ := json.Marshal(errorMap)
		sendErr := debugConn.WriteMessage(websocket.TextMessage,
			errorBytes)
		if sendErr != nil { closeConn(debugConn, debuggerId); return sendErr }
		return nil
	}

	for true {
		_, msgBytes, err := debugConn.ReadMessage()
		if err != nil { closeConn(debugConn, debuggerId); return }

		var msg map[string]string
		err = json.Unmarshal(msgBytes, &msg)
		if err != nil {
			err = sendErrorBytes(fmt.Errorf("Could not unmarshal message."), "")
			if err != nil { return }
			continue
		}
		cmd, ok := msg["command"]

		// message must have a command
		if !ok {
			err = sendErrorBytes(fmt.Errorf("Message must be given a command"), "")
			if err != nil { return }
			continue
		}

		switch cmd {
		case "setBreakpoint":
			retBytes, err := implSetBreakpoint(msg)
			if err != nil {
				err = sendErrorBytes(err, cmd)
				if err != nil { return }
				continue
			}
			err = sendAll(retBytes, debuggerId)
			if err != nil {
				return
			}
		case "unsetBreakpoint":
			retBytes, err := implUnsetBreakpoint(msg)
			if err != nil {
				err = sendErrorBytes(err, cmd)
				if err != nil { return }
				continue
			}
			err = sendAll(retBytes, debuggerId)
			if err != nil {
				return
			}
		case "unregister":
			closeConn(debugConn, debuggerId)
			return
		case "pause":
			retBytes, err := implPause(msg)
			if err != nil {
				err = sendErrorBytes(err, cmd)
				if err != nil { return }
				continue
			}
			err = sendAll(retBytes, debuggerId)
			if err != nil {
				return
			}
		case "unpause":
			retBytes, err := implUnpause(msg)
			if err != nil {
				err = sendErrorBytes(err, cmd)
				if err != nil { return }
				continue
			}
			err = sendAll(retBytes, debuggerId)
			if err != nil {
				return
			}
		}
	}
}

// returns reply message, err
func implSetBreakpoint(bodyJson map[string]string) ([]byte, error) {
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

	var err error

	breakpointId = "bk-"+uuid.New().String()

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
	node, ok = bodyJson["node"]
	if ok { msg["node"] = node }

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
	return msgBytes, nil

errorEncountered:
	return nil, fmt.Errorf("failed to set breakpoint: %v", err)
}

func implUnsetBreakpoint(bodyJson map[string]string) ([]byte, error) {
	var err error

	breakpointId, ok := bodyJson["breakpointId"]
	if !ok {
		err = fmt.Errorf("argument breakpointId not given")
		return nil, err
	}

	var msg = map[string]string {
		"command": "unsetBreakpoint",
		"breakpointId": breakpointId,
	}
	node, ok := bodyJson["node"]
	if ok { msg["node"] = node }

	msgBytes, err := json.Marshal(msg)
	if err != nil { return nil, err }

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

	if !ok || node == "" {
		sidecars, _ := rpc.GetNodeIDs()
		for _, sidecar := range sidecars {
			// TODO: error handling
			// right now, errors might leave
			// nodes in inconsistent state w.r.t.
			// breakpoints

			// TODO: parallelize rpcs

			err = doCall(sidecar)
			if err != nil { return nil, err }
		}
	} else {
		err = doCall(node)
		if err != nil { return nil, err }
	}
	// assume no error at this point
	return msgBytes, nil
}

func routeImplRegisterDebugger(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	isDebugger = true

	// register debugger with all sidecars
	var msg = map[string]string {
		"command": "registerDebugger",
		"nodeId": rpc.GetNodeID(),
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

	var header = r.Header
	debuggerIdList, ok := header["Id"]
	if !ok { fmt.Println("no id found"); fmt.Println(header); return }
	if len(debuggerIdList) < 1 { fmt.Println("no id found"); return }
	debuggerId := debuggerIdList[0]

	debugConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil { fmt.Println(err); return }

	// create the websocket connection
	debugConnsLock.Lock()
	debugConns[debuggerId] = debugConn
	debugConnsLock.Unlock()

	debugServe(debugConn, debuggerId)
}

func unregisterDebuggerSidecar() {
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
		doTell(sidecar)
	}
}


func implPause(bodyJson map[string]string) ([]byte, error){
	var err error

	_, actorTypeOk := bodyJson["actorType"]
	_, actorIdOk := bodyJson["actorId"]

	if !actorTypeOk && actorIdOk {
		err = fmt.Errorf("if actorId is set, then actorType must be set")
		return nil, err
	}

	var msg = map[string]string {
		"command": "pause",
		"actorType": mapget(bodyJson, "actorType", ""),
		"actorId": mapget(bodyJson, "actorId", ""),
	}

	node, ok := bodyJson["node"]
	if ok { msg["node"] = node }

	msgBytes, err := json.Marshal(msg)
	if err != nil { return nil, err }

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

	if !ok || node == "" {
		sidecars, _ := rpc.GetNodeIDs()
		for _, sidecar := range sidecars {
			// TODO: error handling
			// right now, errors might leave
			// nodes in inconsistent state w.r.t.
			// breakpoints

			// TODO: parallelize rpcs

			err = doCall(sidecar)
			if err != nil { return nil, err }
		}
	} else {
		err = doCall(node)
		if err != nil { return nil, err }
	}
	return msgBytes, nil
}

func implUnpause(bodyJson map[string]string) ([]byte, error){
	var err error

	_, actorTypeOk := bodyJson["actorType"]
	_, actorIdOk := bodyJson["actorId"]

	if !actorTypeOk && actorIdOk {
		err = fmt.Errorf("if actorId is set, then actorType must be set")
		return nil, err
	}

	node, ok := bodyJson["node"]

	var msg = map[string]string {
		"command": "unpause",
		"actorType": mapget(bodyJson, "actorType", ""),
		"actorId": mapget(bodyJson, "actorId", ""),
	}
	if ok { msg["node"] = node }

	msgBytes, err := json.Marshal(msg)
	if err != nil { return nil, err }

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

	if !ok || node == "" {
		sidecars, _ := rpc.GetNodeIDs()
		for _, sidecar := range sidecars {
			// TODO: error handling
			// right now, errors might leave
			// nodes in inconsistent state w.r.t.
			// breakpoints

			// TODO: parallelize rpcs

			err = doCall(sidecar)
			if err != nil { return nil, err }
		}
	} else {
		err = doCall(node)
		if err != nil { return nil, err }
	}
	// assume no error at this point
	return msgBytes, nil
}
