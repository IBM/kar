package runtime

import (
	"encoding/json"
	//"context"
	"net/http"
	"fmt"
	"time"
	"strings"

	"github.com/IBM/kar/core/pkg/rpc"
	"github.com/IBM/kar/core/internal/config"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"	
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

type listBreakpointInfo_t struct {
	BreakpointId string `json:"breakpointId"`
	BreakpointType string `json:"breakpointType"`

	ActorType string `json:"actorType"`
	ActorId string `json:"actorId"`
	Path string `json:"path"`

	IsRequest string `json:"isRequest"`

	Nodes []string `json:"nodes"`
}

type listPauseInfo_t struct {
	ActorType string `json:"actorType"`
	ActorId string `json:"actorId"`
	RequestId string `json:"requestId"`
	RequestValue string `json:"requestValue"`
	ResponseValue string `json:"responseValue"`
	IsResponse string `json:"isResponse"`
	BreakpointId string `json:"breakpointId"`
	NodeId string `json:"nodeId"`
}

type actor_t struct {
	// stupid duplicated actor struct
	// TODO: refactor
	ActorType string `json:"actorType"`
	ActorId string `json:"actorId"`
}

// begin indirect pause detection types
type actorSentInfo_t struct {
	Actor actor_t `json:"actor"`
	ParentId string `json:"parentId"`
	RequestValue string `json:"requestValue"`
}

type listBusyInfo_t struct {
	ActorHandling map[string]actor_t `json:"actorHandling"`
	ActorSent map[string]actorSentInfo_t `json:"actorSent"`
}

// end indirect pause detection types

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
		/*err = */json.Unmarshal(msgBytes, &msg)
		/*if err != nil {
			fmt.Println(string(msgBytes))
			fmt.Println(err)
			err = sendErrorBytes(fmt.Errorf("Could not unmarshal message."), "")
			if err != nil { return }
			continue
		}*/
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
		case "listBreakpoints":
			doCallMsg := map[string]string {
				"command": "listBreakpoints",
			}
			var doCallMsgBytes []byte
			doCallMsgBytes, _ = json.Marshal(doCallMsg)

			var listBreakpointsMap = map[string]listBreakpointInfo_t {}
			doCall := func(sidecar string) error {
				//fmt.Println("doing call to "+sidecar)
				bytes, err := rpc.Call(ctx, rpc.Destination{Target: rpc.Node{ID: sidecar}, Method: sidecarEndpoint}, time.Time{}, "", doCallMsgBytes)
				if err != nil { return err }
				var reply Reply
				err = json.Unmarshal(bytes, &reply)
				if err != nil { return err }
				if reply.StatusCode != 200 {
					err = fmt.Errorf("Status code of reply not OK: %v", reply.StatusCode)
					return err
				}
				var payload = map[string]listBreakpointInfo_t {}
				err = json.Unmarshal([]byte(reply.Payload), &payload)
				if err != nil { return err }

				for id, bk := range payload {
					_, found := listBreakpointsMap[id]
					bk.BreakpointId = id
					if !found {
						bk.Nodes = []string{sidecar}
						listBreakpointsMap[id] = bk
					} else {
						newBk := listBreakpointsMap[id]

						nodesList := newBk.Nodes
						nodesList = append(nodesList, sidecar)
						newBk.Nodes = nodesList
						listBreakpointsMap[id] = newBk
					}
				}

				return nil
			}

			var err error

			sidecars, _ := rpc.GetNodeIDs()
			successful := false
			for _, sidecar := range sidecars {
				if /*sidecar != rpc.GetNodeID()*/ true {
					// TODO: parallelize rpcs

					err = doCall(sidecar)
					if err == nil { successful = true }
				}
			}

			if !successful {
				err = sendErrorBytes(err, cmd)
				if err != nil { return }
				continue
			}

			retMsg := map[string]interface{} {}
			retMsg["breakpoints"] = listBreakpointsMap
			retMsg["command"] = "listBreakpoints"

			retBytes, err := json.Marshal(retMsg)

			if err != nil {
				err = sendErrorBytes(err, cmd)
				if err != nil { return }
				continue
			}
			err = sendAll(retBytes, debuggerId)
			if err != nil {
				return
			}
		case "listPausedActors":
			doCallMsg := map[string]string {
				"command": "listPausedActors",
			}
			var doCallMsgBytes []byte
			doCallMsgBytes, _ = json.Marshal(doCallMsg)

			var actorsList = []listPauseInfo_t {}
			doCall := func(sidecar string) error {
				bytes, err := rpc.Call(ctx, rpc.Destination{Target: rpc.Node{ID: sidecar}, Method: sidecarEndpoint}, time.Time{}, "", doCallMsgBytes)
				if err != nil { return err }
				var reply Reply
				err = json.Unmarshal(bytes, &reply)
				if err != nil { return err }
				if reply.StatusCode != 200 {
					err = fmt.Errorf("Status code of reply not OK: %v", reply.StatusCode)
					return err
				}
				var payload = []listPauseInfo_t {}
				err = json.Unmarshal([]byte(reply.Payload), &payload)
				if err != nil { return err }

				for _, info := range payload {
					info.NodeId = sidecar
					actorsList = append(actorsList, info)
				}

				return nil
			}

			var err error

			sidecars, _ := rpc.GetNodeIDs()
			successful := false
			for _, sidecar := range sidecars {
				if /*sidecar != rpc.GetNodeID()*/ true {
					// TODO: parallelize rpcs

					err = doCall(sidecar)
					if err == nil { successful = true }
				}
			}

			if !successful {
				err = sendErrorBytes(err, cmd)
				if err != nil { return }
				continue
			}

			retMsg := map[string]interface{} {}
			retMsg["actorsList"] = actorsList
			retMsg["command"] = "listPausedActors"

			retBytes, err := json.Marshal(retMsg)

			if err != nil {
				err = sendErrorBytes(err, cmd)
				if err != nil { return }
				continue
			}
			err = sendAll(retBytes, debuggerId)
			if err != nil {
				return
			}
		case "listBusyActors":
			doCallMsg := map[string]string {
				"command": "listBusyActors",
			}
			var doCallMsgBytes []byte
			doCallMsgBytes, _ = json.Marshal(doCallMsg)

			var myBusyInfo = listBusyInfo_t {}
			myBusyInfo.ActorHandling = map[string]actor_t {}
			myBusyInfo.ActorSent = map[string]actorSentInfo_t {}
			doCall := func(sidecar string) error {
				bytes, err := rpc.Call(ctx, rpc.Destination{Target: rpc.Node{ID: sidecar}, Method: sidecarEndpoint}, time.Time{}, "", doCallMsgBytes)
				if err != nil { return err }
				var reply Reply
				err = json.Unmarshal(bytes, &reply)
				if err != nil { return err }
				if reply.StatusCode != 200 {
					err = fmt.Errorf("Status code of reply not OK: %v", reply.StatusCode)
					return err
				}
				var payload = listBusyInfo_t {}
				err = json.Unmarshal([]byte(reply.Payload), &payload)
				if err != nil { return err }

				for req, actor := range payload.ActorHandling {
					myBusyInfo.ActorHandling[req]=actor
				}

				for req, sentInfo := range payload.ActorSent {
					myBusyInfo.ActorSent[req]=sentInfo
				}

				return nil
			}

			var err error

			sidecars, _ := rpc.GetNodeIDs()
			successful := false
			for _, sidecar := range sidecars {
				if /*sidecar != rpc.GetNodeID()*/ true {
					// TODO: parallelize rpcs

					err = doCall(sidecar)
					if err == nil { successful = true }
				}
			}

			if !successful {
				err = sendErrorBytes(err, cmd)
				if err != nil {
					return
				}
				continue
			}

			retMsg := map[string]interface{} {}
			retMsg["busyInfo"] = myBusyInfo
			retMsg["command"] = "listBusyActors"
			retMsg["commandId"] = msg["commandId"]

			retBytes, err := json.Marshal(retMsg)

			if err != nil {
				err = sendErrorBytes(err, cmd)
				if err != nil { return }
				continue
			}
			err = sendAll(retBytes, debuggerId)
			if err != nil {
				return
			}
		// below here are KAR commands that you'd normally run from the command line
		// why is this better? because no need to start up a new sidecar and do reconciliation
		case "kar purge":
			purge("kar"+config.Separator+config.AppName, "*")
		case "kar drain":
			purge("kar"+config.Separator+config.AppName,
			"pubsub"+config.Separator+"*")
		case "kar invoke":
			argsMap := map[string][]string{}
			//fmt.Println(msgBytes)
			/*err := */json.Unmarshal(msgBytes, &argsMap)
			/*if err != nil {
				fmt.Println("counldn't unmasrhak")
				err = sendErrorBytes(err, cmd)
				if err != nil { return }
				continue
			}*/
			args, ok := argsMap["args"]
			if !ok {
				err = fmt.Errorf("Error: no arguments provided.")
				err = sendErrorBytes(err, cmd)
				if err != nil { return }
				continue
			}
			if len(args) < 3 {
				err = fmt.Errorf("Error: too few arguments provided.")
				err = sendErrorBytes(err, cmd)
				if err != nil { return }
				continue
			}
			actor := Actor{Type: args[0], ID: args[1]}
			path := "/" + args[2]
			params := make([]interface{}, len(args[3:]))
			for i, a := range args[3:] {
				if json.Unmarshal([]byte(a), &params[i]) != nil {
					params[i] = args[3+i]
					// assuming each arg is a string
				}
			}
			payload, err := json.Marshal(params)
			if err != nil {
				//fmt.Printf("cound't marshal")
				err = sendErrorBytes(err, cmd)
				if err != nil { return }
				continue
			}

			doInvoke := func () {
				reply, err := CallActor(ctx, actor, path, string(payload), "", "") 
				if err != nil {
					//fmt.Printf("condlt call acor")
					err = sendErrorBytes(err, cmd)
					return
				}
				retMsg := map[string]interface{} {}
				retMsg["commandId"] = msg["commandId"]
				retMsg["command"] = "kar invoke"
				retMsg["status"] = reply.StatusCode
				if reply.StatusCode == http.StatusOK {
					if strings.HasPrefix(reply.ContentType, "application/kar+json") {
						var result actorCallResult
						err = json.Unmarshal([]byte(reply.Payload), &result)
						//fmt.Println(reply.Payload)
						if err == nil {
							if result.Error {
								retMsg["error"] = result.Message
							} else {
								retMsg["value"] = result.Value
							}
						}
					} else {
						retMsg["value"] = reply.Payload
					}
				}

				retBytes, err := json.Marshal(retMsg)

				if err != nil {
					err = sendErrorBytes(err, cmd)
					return
				}
				err = sendAll(retBytes, debuggerId)
				if err != nil {
					return
				}
			}
			go doInvoke()
		case "kar rest":
			argsMap := map[string][]string{}
			//fmt.Println(msgBytes)
			json.Unmarshal(msgBytes, &argsMap)
			args, ok := argsMap["args"]
			if !ok {
				err = fmt.Errorf("Error: no arguments provided.")
				err = sendErrorBytes(err, cmd)
				if err != nil { return }
				continue
			}
			if len(args) < 3 {
				err = fmt.Errorf("Error: too few arguments provided.")
				err = sendErrorBytes(err, cmd)
				if err != nil { return }
				continue
			}

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
				//fmt.Printf("service call acor")
				err = sendErrorBytes(err, cmd)
				if err != nil { return }
				continue
			}

			retMsg := map[string]interface{} {}
			retMsg["commandId"] = msg["commandId"]
			retMsg["command"] = "kar rest"
			retMsg["status"] = reply.StatusCode

			if reply.StatusCode != http.StatusOK {
				retMsg["error"] = reply.Payload
			} else {
				retMsg["value"] = reply.Payload
			}

			retBytes, err := json.Marshal(retMsg)
			if err != nil {
				err = sendErrorBytes(err, cmd)
				if err != nil { return }
				continue
			}
			err = sendAll(retBytes, debuggerId)
			if err != nil {
				return
			}
		case "kar get":
			//fmt.Println("get!")
			//fmt.Println(msg)
			switch msg["subsystem"]{
			case "sidecars", "sidecar":
				retMsg := map[string]string {}
				retMsg["commandId"] = msg["commandId"]
				retMsg["command"] = "kar get"
				retMsg["subsystem"] = "sidecars"
				retMsg["sidecars"] = implGetSidecars()

				retBytes, err := json.Marshal(retMsg)
				if err != nil {
					err = sendErrorBytes(err, cmd)
					if err != nil { return }
					continue
				}
				err = sendAll(retBytes, debuggerId)
				if err != nil {
					return
				}

			case "actor", "actors":
				retMsg := map[string]string {}
				retMsg["commandId"] = msg["commandId"]
				retMsg["command"] = "kar get"
				retMsg["subsystem"] = "actors"

				actorId := msg["actorId"]
				actorType := msg["actorType"]
				if actorId != "" {
					if actorState, err := actorGetAllState(actorType, actorId); err == nil {
						if len(actorState) != 0 {
							if bytes, err := json.Marshal(actorState); err == nil {
								retMsg["state"] = string(bytes)
							}
						}
					}
				} else {
					var bytes []byte
					if true /*!config.GetResidentOnly*/ {
						if actorMap, err := rpc.GetAllSessions(ctx, actorType); err == nil {
							bytes, err = json.Marshal(actorMap)
						}
					} else {
						if actorMap, err := getAllActiveActors(ctx, actorType); err == nil {
							bytes, err = json.Marshal(actorMap)
						}
					}
					if err == nil {
						retMsg["actors"] = string(bytes)
					}
				}

				retBytes, err := json.Marshal(retMsg)
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
}

func implGetSidecars() string {
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
	m, _ := json.Marshal(karTopology)
	return string(m)
}

// returns reply message, err
func implSetBreakpoint(bodyJson map[string]string) ([]byte, error) {
	// golang can be very annoying sometimes
	// forward declaration of vars to make goto work
	var breakpointId string
	var msg map[string]interface{}
	var msgBytes []byte
	var doCall func(string) error
	var node string
	var ok bool
	var actorType string
	var path string
	var successful bool
	nodes := []string{}

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

	msg = map[string]interface{} {
		"command": "setBreakpoint",
		"breakpointId": breakpointId,
		"breakpointType": mapget(bodyJson, "breakpointType", "global"),
		"actorId": mapget(bodyJson, "actorId", ""),
		"actorType": actorType,
		"path": path,
		"isCaller": mapget(bodyJson, "isCaller", "caller"),
		"isRequest": mapget(bodyJson, "isRequest", "request"),
		"srcNodeId": rpc.GetNodeID(),
		"deleteOnHit": bodyJson["deleteOnHit"],
		//"nodes": []string{},
	}
	node, ok = bodyJson["node"]
	if ok { msg["node"] = node }

	msgBytes, err = json.Marshal(msg)
	if err != nil { goto errorEncountered }

	doCall = func(sidecar string) error {
		//fmt.Println("doing call to "+sidecar)
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


	successful = false
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
				if err == nil {
					successful = true
					nodes = append(nodes, sidecar)
				}
			}
		}
		if !successful { goto errorEncountered }
	} else {
		err = doCall(node)
		if err != nil { goto errorEncountered }
		nodes = append(nodes, node)
	}
	// assume no error at this point
	msg["nodes"] = nodes
	msg["commandId"] = bodyJson["commandId"]
	msgBytes, err = json.Marshal(msg)
	return msgBytes, err

errorEncountered:
	return nil, fmt.Errorf("failed to set breakpoint: %v", err)
}

func implUnsetBreakpoint(bodyJson map[string]string) ([]byte, error) {
	//fmt.Println("About to impl unset breakpoint.")
	var err error

	breakpointId, ok := bodyJson["breakpointId"]
	if !ok {
		err = fmt.Errorf("argument breakpointId not given")
		return nil, err
	}

	var msg = map[string]interface{} {
		"command": "unsetBreakpoint",
		"breakpointId": breakpointId,
		"srcNodeId": rpc.GetNodeID(),
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

	nodes := []string {}

	if !ok || node == "" {
		sidecars, _ := rpc.GetNodeIDs()
		successful := false
		for _, sidecar := range sidecars {
			// TODO: error handling
			// right now, errors might leave
			// nodes in inconsistent state w.r.t.
			// breakpoints

			// TODO: parallelize rpcs

			err = doCall(sidecar)
			if err == nil {
				successful = true
				nodes = append(nodes, sidecar)
			} else {
				//fmt.Printf("impl breakpoint error: %v\n", err)
			}
		}
		if !successful { return nil, err }
	} else {
		err = doCall(node)
		if err != nil { return nil, err }
		nodes = append(nodes, node)
	}
	// assume no error at this point
	msg["nodes"] = nodes
	msgBytes, err = json.Marshal(msg)
	return msgBytes, err

}

func routeImplRegisterDebugger(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// register debugger with all sidecars
	var msg = map[string]string {
		"command": "registerDebugger",
		"nodeId": rpc.GetNodeID(),
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil { return }

	doTell := func(sidecar string){
		rpc.Tell(ctx, rpc.Destination{Target: rpc.Node{ID: sidecar}, Method: sidecarEndpoint}, time.Time{}, "", msgBytes)
	}

	sidecars, _ := rpc.GetNodeIDs()
	for _, sidecar := range sidecars {
		if true /*sidecar != rpc.GetNodeID()*/ {
			doTell(sidecar)
		}
	}

	var header = r.Header
	debuggerIdList, ok := header["Id"]
	if !ok { /*fmt.Println("no id found"); fmt.Println(header); */return }
	if len(debuggerIdList) < 1 { /*fmt.Println("no id found");*/ return }
	debuggerId := debuggerIdList[0]

	debugConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil { /*fmt.Println(err);*/ return }

	// create the websocket connection
	debugConnsLock.Lock()
	debugConns[debuggerId] = debugConn
	debugConnsLock.Unlock()

	debugServe(debugConn, debuggerId)
	closeConn(debugConn, debuggerId)
}

func unregisterDebuggerSidecar() {
	//isDebugger = false

	// unregister debugger with all sidecars
	var msg = map[string]string {
		"command": "unregisterDebugger",
		"nodeId": rpc.GetNodeID(),
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil { return }

	doTell := func(sidecar string){
		rpc.Tell(ctx, rpc.Destination{Target: rpc.Node{ID: sidecar}, Method: sidecarEndpoint}, time.Time{}, "", msgBytes)
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
		"srcNodeId": rpc.GetNodeID(),
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
		"srcNodeId": rpc.GetNodeID(),
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
