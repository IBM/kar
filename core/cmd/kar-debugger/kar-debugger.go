package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"sync"
	"strings"
	"sort"
	"strconv"
	"encoding/json"
	"crypto/tls"
	"net"
	"github.com/gorilla/websocket"
	"github.com/google/uuid"
	"github.com/Shopify/sarama"
	"time"
	//"log"
)

type sidecarData_t struct {
	Actors   []string `json:"actors"`
	Services []string `json:"services"`
	Host string `json:"host"`
	Port int `json:"port"`
}

var commandUsage = map[string]string {
	"unpause":
`Unpause actors that are paused.
Usage: unpause [actorType actorId] [OPTIONS]

Args:
	actorType:
		The type of actor to be paused. If this argument is ommitted, then all actors on the specified node will be paused. Otherwise, actorId must also be given.
	
	actorId:
		The ID of an actor to be paused. If actorType is ommitted, then this argument must also be ommitted.
	
Options:
	-node nodeId
		the node whose actors should be unpaused
		(default: all nodes)`,
	"b":
`Set a breakpoint.
Usage: b actorType method [OPTIONS]

Args:
	method
		The method that should trigger the breakpoint. Example: "siteReport".
	actorType
		The actor type that should trigger the breakpoint. Example: "Site".
Options:
	-node nodeId
		the node on which the breakpoint should be installed
		(default: all nodes)
	-location [request|response]
		whether the breakpoint should trigger when the actor receives the request or when the actor finishes processing the request
		(default: request)
	-actorId actorId
		the id of the actor that will trigger the breakpoint
		(default: when the actor method is called on any instance of the specified actor type, the breakpoint is triggered)
	-type breakpointType (default: global)
		The type of breakpoint. There are four options. "global" causes all nodes to pause all actors. "node" causes the node that tripped the breakpoint to pause all actors. "actor" causes the actor that tripped the breakpoint to pause. "suicide" causes the node that tripped the breakpoint to die.
	-conds listOfConditions
		A comma-separated list of conditions on each request that
		might trigger the breakpoint. If given, then the
		breakpoint will only trigger if all of the conditions are
		met. Type "help conditions" for more information.
	-respConds listOfConditions
		A comma-separated list of conditions on each response that
		might trigger the breakpoint. If given, then the
		breakpoint will only trigger if all of the conditions are
		met. Type "help conditions" for more information.`,
	"d":
`Delete a breakpoint.
Usage: d breakpointId [OPTIONS]

Args:
	breakpointId
		The ID of the breakpoint to delete.
Options:
	-node nodeId
		The node from which the breakpoint should be deleted
		(default: all nodes)`,

	"vb":
`View information about all breakpoints or a specific breakpoint.
Usage: vb [breakpointId]

Args:
	breakpointId
		The ID of the breakpoint whose information is requested.
		If this argument is omitted, then information about all breakpoints will be displayed.
Options:
	-format format (default: text)
		Determines the output format. If "text", then outputs in a human-readable format. If "json", then outputs in json.`,
	"vpa":
`View a list of paused actors matching certain filters.
Usage: vpa [actorType actorId] [OPTIONS] [FILTERS]

All paused actors that match every given filter are returned. Matches must be exact. If no filters are given, then all paused actors are returned.
For convenience, the actorType and actorId filters can be given as the first two arguments.

Options:
	-format format (default: text)
		Determines the output format. If "text", then outputs in a human-readable format. If "json", then outputs in json.
	-ind isIndirect (default: true)
		If true, then outputs actors that are indirectly paused in addition to actors that are directly paused.
Filters:
	-actorType actorType
		The type of the actor. E.g. "Site".
	-actorId actorId
		The ID of the actor instance. E.g. "42".
	-requestId requestId
		The ID of the request on which the actor is paused.
	-method method
		The method of the request on which the actor is paused. E.g. "doStuff".
	-requestType requestType
		The type of request on which the actor is paused. E.g. "call", "tell".
	-isResponse isResponse
		Whether the actor is paused on a request or a response. E.g. "request", "response".
	-breakpointId breakpointId
		The ID of the breakpoint on which the actor is paused.
	-nodeId nodeId
		The ID of the node on which the actor is paused.`,
	"kar":
`Access kar commands.
Usage: kar subcommand ARGS

Subcommands:
	invoke:
		Invoke an actor method.
	rest:
		Invoke a service method.
	get:
		Get information about sidecars or actors.
`,
	"kar invoke":
`Invoke an actor method.
Usage: kar invoke actorType actorId path [ARGS]

Args:
	actorType:
		The type of actor to invoke.
	actorId:
		The id of the actor to invoke.
	path:
		The path of the actor method to invoke. This path should NOT contain a leading "/".
	ARGS:
		The arguments to be passed to the method. Currently, all arguments are treated as strings.
`,

	"kar rest":
`Call a service method.
Usage: kar rest method serviceName path [ARGS]

Args:
	method:
		The HTTP method (e.g. GET, POST, etc.) by which the service should be accessed.
	serviceName:
		The name of the service to invoke.
	path:
		The path of the service method to invoke. This path should NOT contain a leading "/".
	ARGS:
		The arguments to be passed to the method. Currently, all arguments are treated as strings.
`,

"kar get":
`Get information about actors or sidecars.
Usage: kar get subsystem [actorType actorId]

Args:
	subsystem:
		The kind of subsystem about which information is requested. If subsystem is "actor" or "actors", then information about actors is obtained. If subsystem is "sidecar" or "sidecars", then information about all sidecars is obtained. 
	actorType:
		Only used if subsystem is "actor" or "actors". Denotes the type of actor to get the state of.
	actorId:
		Only used if subsystem is "actor" or "actors". Denotes the id of the actor to get the state of.
`,
"server":
`Start the debugger server, which connects to the KAR cluster
and enables debugging.
Usage: server [karHost karPort]

Args:
	karHost:
		The hostname of the KAR node to which the debugger should connect.
		If the server is launched as a KAR process, this argument is optional.
	karPort:
		The port of the KAR node to which the debugger should connect.
		If the server is launched as a KAR process, this argument is optional.
Options:
	-serverPort port:
		The port on which the debugger server should listen. By default,
		5364.
`,
"step":
`Sets a breakpoint that is triggered when a paused actor finishes
processing the request on which it is paused. Unpauses all actors,
allowing the system to run until this breakpoint is hit.
Usage: step actorType actorId

Args:
	actorType:
		The type of the actor on which to step.
	actorId:
		The id of the actor on which to step.`,
"conditions":
`Information on breakpoint conditions:

It is possible to set finer-grained conditions that must be met for
breakpoints to trigger. For instance, you might only want a breakpoint
to trigger if a method is invoked as a "call", or if the response from
the method is greater than 7. Breakpoint conditions let you achieve this.

All conditions are of the form "arg1 COND arg2", where COND is one of the
following operators: ==, !=, <=, >=, <, >, LIKE, IN. The operators have
the following semantics:
	"==": True if arg1 equals arg2, false otherwise.

	"!=": False if arg1 equals arg2, true otherwise.

	"<=", ">=", "<", ">". False if either argument is not a number.
	True if arg1 is less than or equal to, greater than or equal to,
	less than, greater than arg2 respectively.

	"LIKE": False if either argument is not a string. True if arg1
	matches the regex pattern given in arg2.

	"IN": If both arguments are strings, then returns true if arg1
	is a substring of arg2 and false otherwise.
	If arg2 is a map, then returns true if arg1 is a key in arg2 and
	false otherwise.
	If arg2 is a list, then returns true if arg1 is an element in arg1
	and false otherwise.

Conditions are evaluated on various properties of the request or response.
To access these properties, the following syntax is used:
	.property: accesses the key "property" of the parent map.
	[i]: accesses the i-th array element of the parent.
These selectors are then chained to form more complex expressions. For
example, if we are evaluating a condition on the request, and the request
has the following arguments:
	[ {"hello": "world"}, {"foo": "bar"} ]
then the selector
	.payload[1].foo
would select "bar".

In the above example, where did "payload" come from? The answer: requests
and responses have special "built-in" properties.
Request properties:
	"payload": an array of arguments.
	"command": the type of method invocation. In most cases, either
		"call" or "tell".
	"path": the name of the method being called, with a forward slash
		at the beginning.

Response properties:
	"payload": a map containing a single item, with key "value".
		.payload.value yields the return value of the response.
	"StatusCode": the HTTP status code returned by the request.`,

`vd`:
`View deadlocks.
Checks for deadlocks in the running application. If any are present, prints
out information about the deadlock. 

Usage: vd`,
}

func getArgs(args []string, names []string, options map[string]string, boolOptions map[string]string, startIndex int) map[string]string {
	retval := map[string]string {}

	retval["commandId"] = uuid.New().String()

	isGettingOption := false
	gettingOption := ""
	argIdx := 0
	for i := 0; i < len(args); i++ {
		curSplit := args[i]
		if curSplit == "" { continue }

		if isGettingOption {
			retval[gettingOption] = curSplit
			isGettingOption = false
			continue
		}

		dstOption, ok := options[curSplit]
		if ok {
			// current argument is an option
			if i >= len(args)-1 { break }
			if dstOption == "" {
				// options begin with leading "-" by default
				dstOption = curSplit[1:]
			}
			isGettingOption = true
			gettingOption = dstOption
			continue
		}
		boolOption, ok := boolOptions[curSplit]
		if ok {
			// current option is a boolean option
			if boolOption == "" {
				boolOption = curSplit[1:]
			}
			retval[boolOption] = "true"
			continue
		}

		if argIdx-startIndex >= len(names) { continue }

		if argIdx >= startIndex {
			retval[names[argIdx-startIndex]] = curSplit
		}
		argIdx++
	}
	return retval
}

func getArgsList(args []string, options map[string]string, boolOptions map[string]string, startIndex int) []string {
	retval := []string {}

	isGettingOption := false
	argIdx := 0
	for i := 0; i < len(args); i++ {
		curSplit := args[i]
		if curSplit == "" { continue }

		if isGettingOption {
			continue
		}

		_, ok := options[curSplit]
		if ok {
			// current argument is an option
			if i >= len(args)-1 { break }
			isGettingOption = true
			continue
		}
		_, ok = boolOptions[curSplit]
		if ok {
			// current option is a boolean option
			continue
		}

		if argIdx >= startIndex {
			retval = append(retval, curSplit)
		}
		argIdx++
	}
	return retval
}


func printInfo(s string, x ...interface{}){
	fmt.Printf("[")
	fmt.Printf(time.Now().Format(time.StampMicro))
	fmt.Printf("] ")
	fmt.Printf(s, x...)
}

var (
	//TODO: refactor. this is the websocket connection, not the debugger connection
	conn *websocket.Conn
	sendLock = sync.Mutex{}

	// this next datastructure maps the id of a request to a debugger connection -- very bad naming
	idToConnLock = sync.Mutex{}
	idToConn = map[string]net.Conn {}
)

func send(str string) error {
	sendLock.Lock()
	defer sendLock.Unlock()
	err := conn.WriteMessage(websocket.TextMessage, []byte(str))
	if err != nil {
		conn.Close()
		fmt.Printf("Error sending message: %v\n", err)
		return err
	} else if verbose >= 2{
		fmt.Printf("String %s successfully sent\n", str)
	}
	return nil
}

var registerUrl string

type actor_t struct {
	actorType string
	actorId string
}

type actorJson_t struct {
	// stupid duplicated actor struct
	// TODO: refactor
	ActorType string `json:"actorType"`
	ActorId string `json:"actorId"`
}

type callInfo_t struct {
	actor actor_t
	child *actor_t
	visited bool
}

// begin indirect pause detection types
type actorSentInfo_t struct {
	Actor actorJson_t `json:"actor"`
	ParentId string `json:"parentId"`
	RequestValue string `json:"requestValue"`
	FlowId string `json:"flowId"`
	IsHandling bool `json:"isHandling"`
	isVisited bool
}

type listBusyInfo_t struct {
	ActorHandling map[string]actorSentInfo_t `json:"actorHandling"`
	ActorSent map[string]actorSentInfo_t `json:"actorSent"`
}

type listPauseInfo_t struct {
	ActorType string `json:"actorType"`
	ActorId string `json:"actorId"`
	RequestId string `json:"requestId"`
	FlowId string `json:"flowId"`
	RequestValue string `json:"requestValue"`
	ResponseValue string `json:"responseValue"`
	IsResponse string `json:"isResponse"`
	BreakpointId string `json:"breakpointId"`
	NodeId string `json:"nodeId"`

	IsPaused bool `json:"isPaused"`
	PauseDepth int `json:"pauseDepth"`
	EndActorId string `json:"endActorId"`
	EndActorType string `json:"endActorType"`

	ChildActorId string `json:"childActorId"`
	ChildActorType string `json:"childActorType"`

	CanStep bool `json:"canStep"`
}

func unpackRequestValue(s string) (map[string]interface{}, error) {
	retval := map[string]interface{} {}
	err := json.Unmarshal([]byte(s), &retval)
	payload, ok := retval["payload"]
	if ok {
		payloadStr, ok := payload.(string)
		if ok {
			var payloadMap interface{}//:= map[string]interface{} {}
			err = json.Unmarshal([]byte(payloadStr), &payloadMap)
			retval["payload"] = payloadMap
		}
	} 
	return retval, err
}

func unpackResponseValue(s string) (map[string]interface{}, error) {
	retval := map[string]interface{} {}
	err := json.Unmarshal([]byte(s), &retval)
	payload, ok := retval["Payload"]
	if ok {
		payloadStr, ok := payload.(string)
		if ok {
			//payloadMap := map[string]interface{} {}
			var payloadMap interface{}
			err = json.Unmarshal([]byte(payloadStr), &payloadMap)
			retval["payload"] = payloadMap
		}
	}
	return retval, err
}

type breakpoint_t struct {
	BreakpointId string `json:"breakpointId"`
	BreakpointType string `json:"breakpointType"`

	ActorType string `json:"actorType"`
	ActorId string `json:"actorId"`
	Path string `json:"path"`
	FlowId string `json:"flowId"`

	IsRequest string `json:"isRequest"`

	Nodes map[string]struct{}
	NodesList []string `json:"nodes"`

	DeleteOnHit string `json:"deleteOnHit"`

	HitActorType string
	HitActorId string

	NumPausedActors int

	isPaused bool
}

type kafkaReq_t struct {
	RequestId string
	ParentId string

	RequestType string

	MsgFound bool
	EarliestTailCall reqMsg_t
	LatestTailCall reqMsg_t

	Response interface{}
	ResponseTimestamp int64
}

type genMsg_t struct {
	IsResponse bool
	RequestId string
	ParentId string
	RequestType string
	Sequence int
	Timestamp int64
	Deadline int64
	ActorId string
	ActorType string
	Path string
	Payload interface{}
	Response interface{}
}

type reqMsg_t struct {
	ActorType string
	ActorId string

	Path string
	Payload interface{}

	Sequence int

	Timestamp int64
	Deadline int64
}

func printKafkaReq(req kafkaReq_t) {
	etc := req.EarliestTailCall
	ltc := req.LatestTailCall
	fmt.Printf("* Request %v ", req.RequestId)
	if req.MsgFound {
		fmt.Printf("of type %v to %v %v\n", req.RequestType, etc.ActorType, etc.ActorId)
	} else {
		fmt.Printf("\n")
	}
	if req.ParentId != "" {
		fmt.Printf("\t* Parent: %v\n", req.ParentId)
	}
	if req.MsgFound {
		if etc.Sequence == ltc.Sequence {
			if len(etc.Path) > 0 {
				fmt.Printf("\t* Method: %v.%v()\n", etc.ActorType, etc.Path[1:])
			}
			pretty, _ := json.MarshalIndent(etc.Payload, "\t\t", " ")
			fmt.Printf("\t* Arguments:\n")
			fmt.Printf("\t\t%s\n", pretty)
			fmt.Printf("\t* Timestamp: %v\n", etc.Timestamp)
			fmt.Printf("\t* Deadline: %v\n", etc.Deadline)
		} else {
			fmt.Printf("\t* Earliest tail call:\n")
			fmt.Printf("\t\t* Target actor: %v %v\n", etc.ActorType, etc.ActorId)
			if len(etc.Path) > 0 {
				fmt.Printf("\t\t* Method: %v.%v\n", etc.ActorType, etc.Path[1:])
			}
			pretty, _ := json.MarshalIndent(etc.Payload, "\t\t\t", " ")
			fmt.Printf("\t\t* Arguments:\n")
			fmt.Printf("\t\t%s\n", pretty)
			fmt.Printf("\t\t* Timestamp: %v\n", etc.Timestamp)
			fmt.Printf("\t\t* Deadline: %v\n", etc.Deadline)
			fmt.Printf("\t\t* Sequence number: %v\n", etc.Sequence)

			fmt.Printf("\t* Latest tail call:\n")
			fmt.Printf("\t\t* Target actor: %v %v\n", ltc.ActorType, ltc.ActorId)
			if len(ltc.Path) > 0 {
				fmt.Printf("\t\t* Method: %v.%v\n", ltc.ActorType, ltc.Path[1:])
			}
			pretty, _ = json.MarshalIndent(ltc.Payload, "\t\t\t", " ")
			fmt.Printf("\t\t* Arguments:\n")
			fmt.Printf("\t\t%s\n", pretty)
			fmt.Printf("\t\t* Timestamp: %v\n", ltc.Timestamp)
			fmt.Printf("\t\t* Deadline: %v\n", ltc.Deadline)
			fmt.Printf("\t\t* Sequence number: %v\n", ltc.Sequence)
		}
	}
	if req.Response != nil {
		fmt.Printf("\n")
		fmt.Printf("\t* Response: \n")
		pretty, _ := json.MarshalIndent(req.Response, "\t\t", " ")
		fmt.Printf("\t\t%s\n", pretty)
	}
}

var (
	kafkaClient sarama.Client
	kafkaConsumer sarama.Consumer
	kafkaTopic string
	kafkaErr error = nil
	kafkaLock = sync.RWMutex {}

	pausedActors = map[actor_t]listPauseInfo_t {}
	// TODO: refactor to use RWMutex
	pausedActorsLock = sync.Mutex {}

	//TODO: refactor this to use respChans
	busyInfo = listBusyInfo_t {}
	busyInfoLock = sync.RWMutex {}
	
	breakpoints = map[string]breakpoint_t {}
	// TODO: refactor to use RWMutex
	breakpointsLock = sync.Mutex {}

	respChans = map[string]chan []byte {}
	// TODO: refactor to use RWMutex
	respChansLock = sync.Mutex {}

	// map id of a step breakpoint to the commandId that set it
	stepBreakpoints = map[string]string {}
	// TODO: refactor to use RWMutex
	stepBreakpointsLock = sync.Mutex {}
)

func printPausedActorFull(pauseInfo listPauseInfo_t){
	fmt.Printf("* %s %s on node %s\n", pauseInfo.ActorType, pauseInfo.ActorId, pauseInfo.NodeId)
	if pauseInfo.PauseDepth > 0 {
		// indirectly paused
		fmt.Printf("\t* Indirectly paused due to actor %s %s\n", pauseInfo.EndActorType, pauseInfo.EndActorId)
		fmt.Printf("\t\t* Pause depth: %v\n", pauseInfo.PauseDepth)
		fmt.Printf("\t* Paused while sending request %v\n", pauseInfo.RequestId)
		fmt.Printf("\t* Target actor: %s %s\n", pauseInfo.ChildActorId, pauseInfo.ChildActorType)
		//goto endHandleRequest
	} else {
		fmt.Printf("\t* Paused while processing ")
		if pauseInfo.IsResponse == "response" {
			fmt.Printf("response")
		} else {
			fmt.Printf("request")
		}
		fmt.Printf(" %s\n", pauseInfo.RequestId)
	}
	{
		var reqInfo = map[string]string {}
		err := json.Unmarshal([]byte(pauseInfo.RequestValue), &reqInfo)
		if err != nil {
			fmt.Printf("Error unmarshalling value: %s\n", err)
			fmt.Printf("Value string: \"%s\"\n", pauseInfo.RequestValue)
			fmt.Printf("Pause info: %+v\n", pauseInfo)
			goto endHandleRequest
		}
		fmt.Printf("\t\t* Request type: %s\n", reqInfo["command"])
		fmt.Printf("\t\t* Request path: %s\n", reqInfo["path"])
		fmt.Printf("\t\t* Request payload: %s\n", reqInfo["payload"])
		if pauseInfo.IsResponse != "response" { goto endHandleRequest }

		responseMap := map[string]interface{} {}
		err = json.Unmarshal([]byte(pauseInfo.ResponseValue), &responseMap)
		if err != nil {
			if verbose >= 2 {
				fmt.Printf("Error unpacking response info: %v\nResponse info: %v\n", err, string(pauseInfo.ResponseValue))
			}
			goto endHandleRequest
		}
		fmt.Printf("\n")
		fmt.Printf("\t\t* Response status code: %v\n", responseMap["StatusCode"])
		payloadMap := map[string]interface{} {}
		err = json.Unmarshal([]byte(responseMap["Payload"].(string)), &payloadMap)
		if err != nil {
			goto endHandleRequest
		}
		fmt.Printf("\t\t* Response value: %v\n", payloadMap["value"])
	}
endHandleRequest:
	fmt.Printf("\t* Paused due to breakpoint %s\n", pauseInfo.BreakpointId)

}

func printBreakpoint(b breakpoint_t) {
	fmt.Printf("* Breakpoint %s:\n", b.BreakpointId)
	fmt.Printf("\t* Breakpoint type: %v\n", b.BreakpointType)
	if b.ActorType != "" {
		fmt.Printf("\t* Break on actor type: %v\n", b.ActorType)
	}
	if b.ActorId != "" {
		fmt.Printf("\t* Break on actor ID: %v\n", b.ActorId)
	}
	if b.Path != "" {
		fmt.Printf("\t* Break on method: %v\n", b.Path)
	}
	fmt.Printf("\t* Break on request vs. response: %v\n", b.IsRequest)

	fmt.Printf("\t* Breakpoint present on nodes:\n")
	for node, _ := range b.Nodes {
		fmt.Printf("\t\t* %s\n", node)
	}

	if b.HitActorType != "" {
		fmt.Printf("\n\t* Breakpoint triggered by actor: %s %s\n",
			b.HitActorType, b.HitActorId)
	}
	fmt.Printf("\t* Number of actors paused due to this breakpoint: %v\n", b.NumPausedActors)
}

// try and connect to Kafka
func kafkaInit(appName string) {
	kafkaLock.Lock()
	defer kafkaLock.Unlock()

	// get config arguments

	brokersStr := os.Getenv("KAFKA_BROKERS")
	if brokersStr == "" {
		kafkaErr = fmt.Errorf("At least one Kafka broker is required. Make sure to set the environment variable KAFKA_BROKERS.")
		return
	}
	brokers := strings.Split(brokersStr, ",")

	user := os.Getenv("KAFKA_USERNAME")
	if user == "" { user = "token" }

	password := os.Getenv("KAFKA_PASSWORD")

	version := os.Getenv("KAFKA_VERSION")
	if version == "" { version = "2.8.1" }
	
	enableTLS := false
	{
		tmp := os.Getenv("KAFKA_ENABLE_TLS") 
		if tmp != "" {
			var err error
			enableTLS, err = strconv.ParseBool(tmp)
			if err != nil {
				kafkaErr = fmt.Errorf("Error parsing KAFKA_ENABLE_TLS \"%v\" as bool", tmp)
				return
			}
		}
	}

	TLSSkipVerify := false
	{
		tmp := os.Getenv("KAFKA_TLS_SKIP_VERIFY") 
		if tmp != "" {
			var err error
			enableTLS, err = strconv.ParseBool(tmp)
			if err != nil {
				kafkaErr = fmt.Errorf("Error parsing KAFKA_TLS_SKIP_VERIFY \"%v\" as bool", tmp)
				return
			}
		}
	}

	// make config

	conf := sarama.NewConfig()
	conf.Version, _ = sarama.ParseKafkaVersion(version)

	if enableTLS {
		conf.Net.TLS.Enable = true
		// TODO support custom CA certificate
		if TLSSkipVerify {
			conf.Net.TLS.Config = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
	}


	if password != "" {
		conf.Net.SASL.Enable = true
		conf.Net.SASL.User = user
		conf.Net.SASL.Password = password
		conf.Net.SASL.Handshake = true
		conf.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	}

	// make client

	var err error
	kafkaClient, err = sarama.NewClient(brokers, conf)

	if err != nil {
		kafkaErr = fmt.Errorf("Error making client: %v", err)
		return
	}

	// make consumer
	kafkaConsumer, err = sarama.NewConsumerFromClient(kafkaClient)
	if err != nil {
		kafkaErr = fmt.Errorf("Error making consumer: %v", err)
		return
	}

	kafkaTopic = "kar_" + appName
}

func listenSidecar(){
	for true {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			conn.Close()
			return
		}
		var msg map[string]string
		json.Unmarshal(msgBytes, &msg)

		cmd, ok := msg["command"]
		if !ok {
			continue
		}

		switch cmd {
		case "appName":
			kafkaInit(msg["appName"])
		case "error":
			idToConnLock.Lock()
			debugConn, ok := idToConn[msg["commandId"]]
			if ok {
				sendDebugger(debugConn, msgBytes)
			} else {
				fmt.Printf("Connection for id %v not found.\n", msg["commandId"])
				fmt.Printf("idToConn: %v\n", idToConn)
			}
			// TODO: can we really safely delete after trying to send a message like this
			if _, ok := msg["keepAlive"]; !ok {
				delete(idToConn, msg["commandId"])
			}
			idToConnLock.Unlock()
		case "unpause":
			myActor := actor_t { actorId: msg["actorId"], actorType: msg["actorType"] }
			pausedActorsLock.Lock()

			actorInfo, ok := pausedActors[myActor]
			if ok && actorInfo.PauseDepth == 0 {
				breakpointsLock.Lock()
				bk, ok := breakpoints[actorInfo.BreakpointId]
				if ok {
					fmt.Println("unpausing")
					bk.isPaused = false
					bk.NumPausedActors--
					if bk.NumPausedActors <= 0 {
						bk.HitActorType = ""
						bk.HitActorId = ""
					}
					breakpoints[actorInfo.BreakpointId] = bk
				}
				breakpointsLock.Unlock()
			}

			delete(pausedActors, myActor)

			if msg["actorId"]=="" && msg["actorType"]=="" {
				for actor, info := range pausedActors {
					if info.NodeId == msg["srcNodeId"] {
						actorInfo, ok := pausedActors[actor]
						if ok && actorInfo.PauseDepth == 0 {
							breakpointsLock.Lock()
							bk, ok := breakpoints[actorInfo.BreakpointId]
							if ok {
								bk.isPaused = false
								bk.NumPausedActors--
								if bk.NumPausedActors <= 0 {
									bk.HitActorType = ""
									bk.HitActorId = ""
								}
								breakpoints[actorInfo.BreakpointId] = bk
							}
							breakpointsLock.Unlock()
						}
						delete(pausedActors, actor)
					}
				}
			}
			pausedActorsLock.Unlock()
			if verbose >= 1 {
				if actorType, ok := msg["actorType"]; ok && actorType != "" {
					fmt.Printf("[NOTICE] %s %s unpaused\n", actorType, msg["actorId"])
				} else {
					fmt.Printf("[NOTICE] Node unpaused\n")
				}
			}
		case "notifyPause":
			handleNotifyPause(msg["actorType"], msg["actorId"], msgBytes, false)
		case "notifyBreakpoint":
			deleteOnHit := false
			
			breakpointsLock.Lock()
			bk, ok := breakpoints[msg["breakpointId"]]

			if ok {
				if bk.DeleteOnHit == "true" {
					deleteOnHit = true
				}

				if deleteOnHit {
					printInfo("Single-step of actor %s %s has paused on response %s\n", msg["actorType"], msg["actorId"], msg["requestId"])
				} else if bk.NumPausedActors == 0 || !bk.isPaused {
					printInfo("Breakpoint %s hit by %s %s with request %s\n", msg["breakpointId"], msg["actorType"], msg["actorId"], msg["requestId"])

					bk.HitActorType = msg["actorType"]
					bk.HitActorId = msg["actorId"]
					bk.isPaused = true
					breakpoints[msg["breakpointId"]] = bk
				}

				if bk.DeleteOnHit == "true" {
					go func(){
						stepBreakpointsLock.Lock()
						defer stepBreakpointsLock.Unlock()

						cmdId, ok := stepBreakpoints[bk.BreakpointId]
						if !ok { return }
						idToConnLock.Lock()
						defer idToConnLock.Unlock()

						debuggerConn, ok := idToConn[cmdId]
						if !ok { return }
						sendDebugger(debuggerConn, msgBytes)
					}()
				}

				//now, actors don't send pause notifications separately from when they hit breakpoints
				//so we handle it for them now
				//note: msgBytes is used in handleNotifyPaused to construct a listPauseInfo_t
				//we can do this because it turns out that most of the fields of
				//  listPauseInfo_t are the same fields as breakpoint_t
				handleNotifyPause(msg["actorType"], msg["actorId"], msgBytes, true)
			}

			breakpointsLock.Unlock()

		case "setBreakpoint":
			breakpointsLock.Lock()
			id := msg["breakpointId"]

			if msg["deleteOnHit"] == "true" {
				// this is a breakpoint set as part of a step cmd
				stepBreakpointsLock.Lock()
				stepBreakpoints[id] = msg["commandId"]
				stepBreakpointsLock.Unlock()
			}

			msgGeneral := map[string]interface{} {}
			json.Unmarshal(msgBytes, &msgGeneral)
			nodes, ok := msgGeneral["nodes"].([]interface{})
			if !ok {
				fmt.Printf("\n\n%+v\n", msgGeneral["nodes"])
				return
			}

			if bk, ok := breakpoints[id]; !ok {
				newBreakpoint := breakpoint_t {
					ActorId: msg["actorId"],
					ActorType: msg["actorType"],
					FlowId: msg["flowId"],
					Path: msg["path"],
					BreakpointId: id,
					BreakpointType: msg["breakpointType"],
					IsRequest: msg["isRequest"],
					Nodes: map[string]struct{}{},
				}
				if msg["deleteOnHit"] == "true" {
					newBreakpoint.DeleteOnHit = "true"
				}
				for _, node := range nodes {
					newBreakpoint.Nodes[node.(string)] = struct{}{}
				}

				fmt.Printf("\r")
				
				if newBreakpoint.DeleteOnHit != "true" {
					printInfo("Breakpoint %s set\n", msg["breakpointId"])
				} else {
					printInfo("Actor %s %s is being single-stepped\n", msg["actorType"], msg["actorId"])
				}
				breakpoints[msg["breakpointId"]] = newBreakpoint
			} else {
				bk.Nodes = map[string]struct{}{}
				for _, node := range nodes {
					bk.Nodes[node.(string)] = struct{}{}
				}
				breakpoints[msg["breakpointId"]] = bk
			}
			breakpointsLock.Unlock()

			breakpointResponse := map[string]string {}
			breakpointResponse["breakpointId"] = id
			breakpointBytes, _ := json.Marshal(breakpointResponse)

			idToConnLock.Lock()
			debugConn, ok := idToConn[msg["commandId"]]
			if ok {
				sendDebugger(debugConn, breakpointBytes)
			} else {
				fmt.Printf("Connection for id %v not found.\n", msg["commandId"])
				fmt.Printf("idToConn: %v\n", idToConn)
			}
			// TODO: can we really safely delete after trying to send a message like this
			if _, ok := msg["keepAlive"]; !ok {
				delete(idToConn, msg["commandId"])
			}
			idToConnLock.Unlock()
		case "unsetBreakpoint":
			breakpointsLock.Lock()
			id := msg["breakpointId"]

			msgGeneral := map[string]interface{} {}
			json.Unmarshal(msgBytes, &msgGeneral)
			nodes := msgGeneral["nodes"].([]interface{})

			if bk, ok := breakpoints[id]; ok {
				for _, node := range nodes {
					delete(bk.Nodes, node.(string))
				}

				if len(bk.Nodes) == 0 {
					delete(breakpoints, id)
				}
				fmt.Printf("\r")
				if bk.DeleteOnHit != "true" {
					printInfo("Breakpoint %s deleted\n", msg["breakpointId"])
				}
			}
			breakpointsLock.Unlock()
		case "listBreakpoints":
			listMsg := map[string]map[string]breakpoint_t {}
			json.Unmarshal(msgBytes, &listMsg)
			bks := listMsg["breakpoints"]
			for id, bk := range bks {
				bk.Nodes = map[string]struct{}{}
				for _, node := range bk.NodesList {
					bk.Nodes[node] = struct{}{}
				}
				bks[id] = bk
			}

			breakpointsLock.Lock()
			breakpoints = bks
			breakpointsLock.Unlock()
		case "listPausedActors":
			listMsg := map[string][]listPauseInfo_t {}
			json.Unmarshal(msgBytes, &listMsg)
			infos := listMsg["actorsList"]

			pausedActorsLock.Lock()
			for _, info := range infos {
				pausedActors[actor_t { actorType: info.ActorType, actorId: info.ActorId }] = info
				
				breakpointsLock.Lock()
				bk, ok := breakpoints[info.BreakpointId]
				if ok && info.PauseDepth == 0 {
					bk.NumPausedActors++
					breakpoints[info.BreakpointId] = bk
				}
				breakpointsLock.Unlock()

			}
			pausedActorsLock.Unlock()
		case "listBusyActors":
			pausedActorsLock.Lock()
			listMsg := map[string]listBusyInfo_t {}
			json.Unmarshal(msgBytes, &listMsg)
			info := listMsg["busyInfo"]

			busyInfoLock.Lock()
			busyInfo = info
			busyInfoLock.Unlock()

			var visitedMap = map[string]bool {}

			for req, curInfo := range info.ActorSent {
				//fmt.Printf("req: %v\n", req)
				if visitedMap[req] { continue }
				// now we're a leaf node

				visitedMap[req] = true

				isPaused := false
				curActor := actor_t {
					actorId: curInfo.Actor.ActorId,
					actorType: curInfo.Actor.ActorType,
				}
				var endInfo listPauseInfo_t
				if tmp, ok := pausedActors[curActor]; ok {
					if tmp.PauseDepth == 0 {
						isPaused = true
						endInfo = tmp

						tmp.IsPaused = true
						tmp.PauseDepth = 0
						tmp.EndActorId = curActor.actorId
						tmp.EndActorType = curActor.actorType
						tmp.FlowId = curInfo.FlowId
						tmp.CanStep = (curInfo.ParentId != "")
						pausedActors[curActor] = tmp
					}
				}

				depth := 1
				curReq := curInfo.ParentId
				childReq := req

				for curReq != "" {
					sentInfo, sentOk := info.ActorSent[curReq]
					if sentOk {
						curActor = actor_t {
							actorId: sentInfo.Actor.ActorId,
							actorType: sentInfo.Actor.ActorType,
						}
					} else {
						handleInfo, ok := info.ActorHandling[curReq]
						if !ok {
							fmt.Printf("Parent request not found in ActorHandling?!\n")
						}
						curActor = actor_t {
							actorId: handleInfo.Actor.ActorId,
							actorType: handleInfo.Actor.ActorType,
						}
					}

					_, pausedOk := pausedActors[curActor]
					if isPaused && !pausedOk {
						tmp := endInfo
						tmp.IsPaused = true
						tmp.PauseDepth = depth
						tmp.ActorId = curActor.actorId
						tmp.ActorType = curActor.actorType
						tmp.RequestId = childReq
						// TODO: is it curInfo.flowId that we should be using? or sentInfo.flowId? all calls in the same call chain have the same flowId, so probably doesn't matter.
						tmp.FlowId = curInfo.FlowId
						tmp.RequestValue = info.ActorSent[childReq].RequestValue
						tmp.ChildActorId = info.ActorSent[childReq].Actor.ActorId
						tmp.ChildActorType = info.ActorSent[childReq].Actor.ActorType

						// all indirectly-paused actors are necessarily paused on request
						// thus, we can always step them
						tmp.CanStep = true
						tmp.IsResponse = "request"

						pausedActors[curActor] = tmp
						//fmt.Printf("tmp: %v %v\n",pausedActors[curActor], curActor) 
					} else {
						//fmt.Printf("npa: %v, %v\n",d, curActor) 
					}
					depth++
					childReq = curReq
					if sentOk {
						curReq = sentInfo.ParentId
					} else {
						curReq = ""
					}
				}
			}
			pausedActorsLock.Unlock()

			respChansLock.Lock()
			c, ok := respChans[msg["commandId"]]
			if ok {
				c<- []byte {}
				delete(respChans, msg["commandId"])
			}
			respChansLock.Unlock()
		/* for certain commands, just send directly to the debugger
		client the response from the sidecar */
		case "kar invoke":
			fallthrough
		case "kar rest":
			fallthrough
		case "kar get":
			idToConnLock.Lock()
			debugConn, ok := idToConn[msg["commandId"]]
			if ok {
				sendDebugger(debugConn, msgBytes)
			}
			// TODO: can we really safely delete after trying to send a message like this
			delete(idToConn, msg["commandId"])
			idToConnLock.Unlock()
		}

		str := string(msgBytes)
		if verbose >= 2 {
			fmt.Printf(":::: Received: %s\n\n", str)
		}
	}
}

func handleNotifyPause(actorType string, actorId string, msgBytes []byte, hasBkLock bool){
	myActor := actor_t { actorId: actorId, actorType: actorType }
	myInfo := listPauseInfo_t {}
	json.Unmarshal(msgBytes, &myInfo)

	pausedActorsLock.Lock()
	_, ok := pausedActors[myActor]

	if !ok {
		if !hasBkLock { breakpointsLock.Lock() }
		bk, ok := breakpoints[myInfo.BreakpointId]
		if ok && myInfo.PauseDepth == 0 {
			bk.NumPausedActors++
			breakpoints[myInfo.BreakpointId] = bk
		}
		if !hasBkLock { breakpointsLock.Unlock() }
	}


	pausedActors[myActor] = myInfo
	pausedActorsLock.Unlock()

	if verbose >= 1 {
		fmt.Printf("[NOTICE] %s %s has been paused\n", actorType, actorId)
	}
}

func sendDebugger(conn net.Conn, msgBytes []byte) error {
	/* TODO:
	Rright now, messages are delimited by a null byte.
	The assumption: no null bytes will ever show up in our
	JSON-formatted messages.

	However, this assumption might not be true.

	Thus, look into a better method of delimiting messages
	*/

	n, err := conn.Write(append(msgBytes, 0))
	if err != nil || n < len(msgBytes)+1 {
		if verbose >= 2 {
			fmt.Printf("Error sending debugger %s: %v\n", string(msgBytes), err)
		}
		return err
	}

	if verbose >= 2 {
		fmt.Printf("Sent debugger %s\n", string(msgBytes))
	}
	return nil
}

// deadlock detection code

type request_t struct {
	RequestId string
	CallStack string // flow ID of request; name is vestigial
	RequestValue string
}

type edge_t struct {
	SrcActor actorJson_t
	DstActor actorJson_t
	Req      request_t
	ParentValue string
}

type node_t struct {
	Edges     map[request_t]edge_t
	IsVisited bool
}

type graph_t = map[actorJson_t]node_t

func makeDeadlockGraph() graph_t {
	busyInfoLock.RLock()
	retgraph := make(graph_t)
	empty := actorJson_t{}

	// TODO: add edges to empty node
	retgraph[empty] = node_t {
		Edges: make(map[request_t]edge_t),
		IsVisited: false,
	}

	// really dumb way of getting flow IDs of root reqs
	// TODO: refactor
	for _, info := range busyInfo.ActorSent {
		_, ok := busyInfo.ActorSent[info.ParentId]
		if !ok {
			// parent is a root req
			parentInfo := busyInfo.ActorHandling[info.ParentId]
			myReq := request_t {
				RequestId: info.ParentId,
				CallStack: info.FlowId, //parent and child have same flow
				RequestValue: parentInfo.RequestValue,
			}
			retgraph[empty].Edges[myReq] = edge_t {
				Req: myReq, DstActor: parentInfo.Actor, SrcActor: empty,
			}

		}
	}

	for reqId, info := range busyInfo.ActorSent {
		srcActor := busyInfo.ActorHandling[info.ParentId].Actor
		dstActor := info.Actor

		parentValue := busyInfo.ActorHandling[info.ParentId].RequestValue

		node, ok := retgraph[srcActor]
		if !ok {
			node = node_t {
				Edges: make(map[request_t]edge_t),
				IsVisited: false,
			}
		}
		myReq := request_t {
			RequestId: reqId,
			CallStack: info.FlowId,
			RequestValue: info.RequestValue,
		}
		edge := edge_t {DstActor: dstActor, Req: myReq, SrcActor: srcActor, ParentValue: parentValue}
		node.Edges[myReq] = edge
		retgraph[srcActor] = node
	}
	busyInfoLock.RUnlock()
	return retgraph
}

func checkDeadlockCycles(g *graph_t, a actorJson_t, path []edge_t) (bool, []edge_t) {
	if (*g)[a].IsVisited {
		for i, edge := range path {
			if edge.DstActor == a {
				req := edge.Req
				//pretty sure that only one of these edges
				// can exist
				if req.CallStack == path[len(path)-1].Req.CallStack {
					//same callstack, so reentrancy, so
					// no deadlock
					return false, nil
				} else {
					// no reentrancy, so deadlock
					return true, path[i:]
				}
			}
		}
		// technically don't need this
		return true, path
	}

	newNode := (*g)[a]
	newNode.IsVisited = true
	(*g)[a] = newNode

	defer func() {
		newNode := (*g)[a]
		newNode.IsVisited = false
		(*g)[a] = newNode
	}()

	for _, edge := range (*g)[a].Edges {
		c, npath := checkDeadlockCycles(g, edge.DstActor, append(path, edge))
		if c {
			return c, npath
		}

	}
	return false, nil
}

func checkDeadlock() []edge_t {
	graph := makeDeadlockGraph()
	_, path := checkDeadlockCycles(&graph, actorJson_t{}, []edge_t{})
	return path
}


// end deadlock detection code

// request info code 

func viewRequestDetails(info actorSentInfo_t, reqId string) {
	fmt.Printf("* Request %s to actor %s %s\n",
		reqId, info.Actor.ActorType, info.Actor.ActorId)
	if info.IsHandling {
		fmt.Printf("\t* Request is currently being handled.\n")
	} else {
		fmt.Printf("\t* Request is in flight, not yet being handled.\n")
	}
	if info.ParentId != "" {
		fmt.Printf("\t* Parent request: %s\n", info.ParentId)
	}
	reqVal, err := unpackRequestValue(info.RequestValue)
	if err != nil { return }
	fmt.Printf("\t* Request type: %s\n", reqVal["command"])
	fmt.Printf("\t* Method: %s()\n", reqVal["path"].(string)[1:])
	pretty, _ := json.MarshalIndent(reqVal["payload"],
		"\t\t", " ")
	fmt.Printf("\t* Arguments:\n%s\n", pretty)
}

// end request info code

func recvDebugger(connReader *bufio.Reader) ([]byte, error) {
	bytes, err := connReader.ReadBytes(0)
	if err != nil { return nil, err}
	return bytes[:len(bytes)-1], nil
}

func processReadKafka(debugMsg map[string]string) /*([]reqMsg_t, error) { //*/(map[string]kafkaReq_t, error) {
	kafkaLock.Lock()
	defer kafkaLock.Unlock()

	if kafkaErr != nil {
		// TODO: send error to debugger
		//fmt.Printf("KafkaErr: %v\n", kafkaErr)
		return nil, kafkaErr
	}

	partitions, err := kafkaConsumer.Partitions(kafkaTopic)
	if err != nil {
		return nil, err
	}


	var partitionConsumers = make(map[int32]sarama.PartitionConsumer)
	for _, partition := range partitions {
		oldest, _ :=
			kafkaClient.GetOffset(kafkaTopic, partition, sarama.OffsetOldest)

		pc, _/*err*/ :=
			kafkaConsumer.ConsumePartition(kafkaTopic,
				partition,
				oldest,
			)
		partitionConsumers[partition] = pc

		// check if any messages to read
		if pc != nil {
			hwo := pc.HighWaterMarkOffset()
			if hwo == oldest {
				// no messages to read
				partitionConsumers[partition] = nil
				continue
			}
		}
	}

	defer func(){
		for _, pc := range partitionConsumers {
			if pc != nil {
				pc.Close()
			}
		}
	}()

	//sarama.Logger = log.New(os.Stderr, "[Sarama] ", log.LstdFlags)

	handleMessage := func(msg sarama.ConsumerMessage) genMsg_t {
		timestamp := msg.Timestamp.Unix()
		var retval = genMsg_t {Timestamp: timestamp}

		for _, header := range msg.Headers {
			key := string(header.Key)
			value := string(header.Value)
			//fmt.Printf("key: %v\n", key)

			// ignore all handlerSidecar methods
			// TODO: right now, this is desired behavior, since
			// handler sidecar methods are just debugging methods
			// (e.g. getActorInformation, etc.)
			// but this behavior might not be desired in the future
			if key == "Method" && value == "handlerSidecar" {
				break
			}

			if key == "RequestID" {
				//fmt.Printf("reqId: %v\n", value)
				retval.RequestId = value
			}

			if key == "Type" {
				if value == "Call" || value == "Tell" {
					retval.IsResponse = false
					retval.RequestType = strings.ToLower(value)
				} else if value == "Response" {
					retval.IsResponse = true
					retval.RequestType = "call"
				} else if value == "Done" {
					retval.IsResponse = true
					retval.RequestType = "tell"
				}
			}

			if key == "Sequence" {
				retval.Sequence, _ = strconv.Atoi(value)
			}

			if key == "Deadline" {
				deadline, _ := strconv.Atoi(value)
				retval.Deadline = int64(deadline)
			}

			if key == "Parent" {
				retval.ParentId = value
			}

			if key == "Session" {
				retval.ActorId = value
			}

			if key == "Service" {
				retval.ActorType = value
			}
			
		}

		if !retval.IsResponse {
			reqVal, err := unpackRequestValue(string(msg.Value))
			if err == nil {
				retval.Path, _ = reqVal["path"].(string)
				retval.Payload = reqVal["payload"]
			}
		} else {
			respVal, err := unpackResponseValue(string(msg.Value))
			if err == nil {
				retval.Response = respVal
			}
		}
		
		return retval
	}

	retmap := map[string]kafkaReq_t {}
	retLock := sync.Mutex {}

	var wg = sync.WaitGroup {}
	loopMsgs := func(partition int32, partitionConsumer sarama.PartitionConsumer){
		defer wg.Done()
		msgs := partitionConsumer.Messages()
		hwo := partitionConsumer.HighWaterMarkOffset()

		for msg := range msgs {
			genMsg := handleMessage(*msg)
			//if genMsg.requestId == debugMsg["requestId"] {
			var genMap = map[string]interface{} {}
			genMsgBytes, _ := json.Marshal(genMsg)
			err := json.Unmarshal(genMsgBytes, &genMap)
			if err != nil {
				continue
			}
			if runConds(genMap, debugMsg["conds"]){
				retLock.Lock()
				curReq, ok := retmap[genMsg.RequestId]
				if ok {
					curReq.RequestType = genMsg.RequestType
					if genMsg.IsResponse {
						curReq.Response = genMsg.Response
						curReq.ResponseTimestamp = genMsg.Timestamp
					} else {
						reqMsg := reqMsg_t {
							ActorType: genMsg.ActorType,
							ActorId: genMsg.ActorId,
							Path: genMsg.Path,
							Payload: genMsg.Payload,
							Sequence: genMsg.Sequence,
							Timestamp: genMsg.Timestamp,
							Deadline: genMsg.Deadline,
						}
						if curReq.EarliestTailCall.Sequence > reqMsg.Sequence || !curReq.MsgFound {
							curReq.MsgFound = true
							curReq.EarliestTailCall = reqMsg
						} else if curReq.LatestTailCall.Sequence < reqMsg.Sequence || !curReq.MsgFound{
							curReq.MsgFound = true
							curReq.LatestTailCall = reqMsg
						}
					}
					retmap[genMsg.RequestId] = curReq
				} else {
					req := kafkaReq_t {
						RequestId: genMsg.RequestId,
					}
					if genMsg.IsResponse {
						req.Response = genMsg.Response
						req.ResponseTimestamp = genMsg.Timestamp
					} else {
						req.RequestType = genMsg.RequestType
						req.ParentId = genMsg.ParentId
						req.MsgFound = true

						reqMsg := reqMsg_t {
							ActorType: genMsg.ActorType,
							ActorId: genMsg.ActorId,
							Path: genMsg.Path,
							Payload: genMsg.Payload,
							Sequence: genMsg.Sequence,
							Timestamp: genMsg.Timestamp,
							Deadline: genMsg.Deadline,
						}

						req.EarliestTailCall = reqMsg
						req.LatestTailCall = reqMsg

					}
					retmap[genMsg.RequestId] = req
				}
				retLock.Unlock()
			}
			if msg.Offset == hwo-1 {
				break
			}
		}
	}

	for partition, partitionConsumer := range partitionConsumers {
		if partitionConsumer == nil { continue }
		wg.Add(1)
		go loopMsgs(partition, partitionConsumer)
	}
	//wg.Add(1)
	//go loopMsgs(0, partitionConsumers[0])

	wg.Wait()

	return retmap, nil
	//return retlist, nil
}

func serveDebugger(conn net.Conn) {
	connReader := bufio.NewReader(conn)
readBytesAgain:
	msgBytes, err := recvDebugger(connReader)
	if err != nil {
		if verbose >= 2 {
			fmt.Println("Error receiving from new debugger connection.")
		}
		return
	}

	msg := map[string]string {}

	/*err = */json.Unmarshal(msgBytes, &msg)
	/*if err != nil {
		fmt.Println("Error unmarshalling from new debugger connection: %v")
		return
	}*/
	
	id, ok := msg["commandId"]
	if !ok {
		if verbose >= 2 {
			fmt.Println("Error getting command id from new debugger message")
		}
		return
	}

	// TODO: better error handling -- if there's an issue, then send an error message back to the client
	cmd, ok := msg["command"]
	if !ok {
		if verbose >= 2 {
			fmt.Println("Error getting command from debugger")
		}
		return
	}

	idToConnLock.Lock()
	idToConn[id] = conn
	idToConnLock.Unlock()

	switch cmd {
	case "vkr":
		// TODO: error handling
		reqMap, _/*err*/ := processReadKafka(msg)
		reqMapBytes, _ := json.Marshal(reqMap)
		sendDebugger(conn, reqMapBytes)
	case "unpause", "setBreakpoint", "unsetBreakpoint",
		"kar invoke", "kar get", "kar rest":
		// just forward it on
		send(string(msgBytes))
	case "viewDeadlocks":
		lbamsg := map[string]string {
			"command": "listBusyActors",
			"commandId": uuid.New().String(),
		}

		respChansLock.Lock()
		respChans[lbamsg["commandId"]] = make(chan []byte)
		respChansLock.Unlock()

		lbamsgBytes, _ := json.Marshal(lbamsg)
		send(string(lbamsgBytes))
		<-respChans[lbamsg["commandId"]]

		path := checkDeadlock()
		responseBytes, _ := json.Marshal(path)
		sendDebugger(conn, responseBytes)
	case "viewRequest":
		lbamsg := map[string]string {
			"command": "listBusyActors",
			"commandId": uuid.New().String(),
		}

		respChansLock.Lock()
		respChans[lbamsg["commandId"]] = make(chan []byte)
		respChansLock.Unlock()

		lbamsgBytes, _ := json.Marshal(lbamsg)
		send(string(lbamsgBytes))
		<-respChans[lbamsg["commandId"]]

		busyInfoLock.RLock()
		defer busyInfoLock.RUnlock()
		reqId := msg["requestId"]
		req, ok := busyInfo.ActorHandling[reqId]
		if ok {
			req.IsHandling = true
		} else {
			req, ok = busyInfo.ActorSent[reqId]
		}

		response := map[string]actorSentInfo_t {}

		if ok {
			response["request"] = req
		}

		responseBytes, _ := json.Marshal(response)
		sendDebugger(conn, responseBytes)
	case "viewBreakpoint":
		breakpointsLock.Lock()
		if bkid, ok := msg["breakpointId"]; ok {
			// send back a single breakpoint

			response := map[string]breakpoint_t {}
			bk, ok := breakpoints[bkid]
			if ok {
				response["breakpoint"] = bk
			}
			responseBytes, _ /*err*/ := json.Marshal(response)
			// TODO: add error checking
			sendDebugger(conn, responseBytes)
		} else {
			response := map[string][]breakpoint_t {}
			bksList := []breakpoint_t {}
			for _, bk := range breakpoints {
				bksList = append(bksList, bk)
			}
			response["breakpoints"] = bksList
			responseBytes, _ /*err*/ := json.Marshal(response)
			// TODO: add error checking
			sendDebugger(conn, responseBytes)
		}
		breakpointsLock.Unlock()
	case "viewPausedActor":
		if msg["ind"] == "true" {
			lbamsg := map[string]string {
				"command": "listBusyActors",
				"commandId": uuid.New().String(),
			}

			respChansLock.Lock()
			respChans[lbamsg["commandId"]] = make(chan []byte)
			respChansLock.Unlock()

			lbamsgBytes, _ := json.Marshal(lbamsg)
			send(string(lbamsgBytes))
			<-respChans[lbamsg["commandId"]]
		}

		var responseBytes []byte
		pausedActorsLock.Lock()
		/*if actorType, ok := msg["actorType"]; ok {
			// send back a single actor
			response := map[string]listPauseInfo_t {}
			if actorId, ok := msg["actorId"]; ok {
				// we have an actor type and actor info
				info, ok := pausedActors[actor_t {
					actorId: actorId,
					actorType: actorType,
				}]
				if ok {
					response["actor"] = info
				}
			}
			responseBytes, _  = json.Marshal(response)
			// TODO: add error checking
		} else { */
			response := map[string][]listPauseInfo_t {}
			pausedList := []listPauseInfo_t {}
			for _, info := range pausedActors {
				// check conditions
				// TODO: fix
				val, ok := msg["actorId"]
				if ok && info.ActorId != val {
					continue
				}

				val, ok = msg["actorType"]
				if ok && info.ActorType != val {
					continue
				}

				val, ok = msg["requestId"]
				if ok && info.RequestId != val {
					continue
				}

				val, ok = msg["isResponse"]
				if ok && info.IsResponse != val {
					continue
				}

				val, ok = msg["breakpointId"]
				if ok && info.BreakpointId != val {
					continue
				}

				val, ok = msg["nodeId"]
				if ok && info.NodeId != val {
					continue
				}

				val, ok = msg["method"]
				if ok {
					var reqInfo = map[string]string {}
					err := json.Unmarshal([]byte(info.RequestValue), &reqInfo)
					if err != nil { continue }
					if reqInfo["path"][1:]!= val { continue }
				}

				val, ok = msg["requestType"]
				if ok {
					var reqInfo = map[string]string {}
					err := json.Unmarshal([]byte(info.RequestValue), &reqInfo)
					if err != nil { continue }
					if reqInfo["command"] != val { continue }
				}

				pausedList = append(pausedList, info)
			}
			response["actors"] = pausedList

			responseBytes, err = json.Marshal(response)
			if err != nil {
				if verbose >= 2 {
					fmt.Printf("Error marshalling response: %v\n", err)
				}
			}
		//}
		pausedActorsLock.Unlock()
		sendDebugger(conn, responseBytes)
	}
	if _, ok := msg["keepAlive"]; ok {
		goto readBytesAgain
	}
}

func processClientKar(karArgs []string, conn net.Conn, connReader *bufio.Reader) {
	if len(karArgs) == 0 {
		fmt.Println(commandUsage["kar"])
		return
	}
	argsMap := getArgs(os.Args, []string{}, map[string]string {
			"-v": "verbose",
		}, map[string]string {
			"-h": "help",
			"-help": "",
		}, 2)
	if argsMap["help"] == "true" {
		fmt.Println(commandUsage["kar " + karArgs[0]])
		return
	}
	switch karArgs[0] {
	case "invoke":
		if len(karArgs) < 4 {
			fmt.Println("Error: too few arguments.")
			fmt.Println(commandUsage["kar invoke"])
			return
		}
		msg := map[string]interface{} {
			"command": "kar invoke",
			"args": karArgs[1:],
			"commandId": uuid.New().String(),
		}
		msgBytes, _ := json.Marshal(msg)
		err := sendDebugger(conn, msgBytes)
		if err != nil {
			fmt.Printf("Error sending message to debugger: %v\n", err)
			return
		}

		/* response: directly from sidecar */
		responseBytes, err := recvDebugger(connReader)
		if err != nil {
			fmt.Printf("Error receiving response from debugger: %v\n", err)
			return
		}

		invokeMsg := map[string]interface{} {}
		json.Unmarshal(responseBytes, &invokeMsg)
		fmt.Printf("\r")
		fmt.Println("Invocation result:")
		fmt.Printf("\t* Status: %v\n", invokeMsg["status"])
		fmt.Printf("\t* Value: %v\n", invokeMsg["value"])
	case "rest":
		if len(karArgs) < 1+3 {
			fmt.Println("Error: too few arguments.")
			fmt.Println(commandUsage["kar rest"])
			return
		}
		msg := map[string]interface{} {
			"command": "kar rest",
			"args": karArgs[1:],
			"commandId": uuid.New().String(),
		}
		msgBytes, _ := json.Marshal(msg)
		err := sendDebugger(conn, msgBytes)
		if err != nil {
			fmt.Printf("Error sending message to debugger: %v\n", err)
			return
		}

		/* response: directly from sidecar */
		responseBytes, err := recvDebugger(connReader)
		if err != nil {
			fmt.Printf("Error receiving response from debugger: %v\n", err)
			return
		}

		restMsg := map[string]interface{} {}
		json.Unmarshal(responseBytes, &restMsg)
		fmt.Println("Service call result:")
		fmt.Printf("\t* Status: %v\n", restMsg["status"])
		fmt.Printf("\t* Error: %v\n", restMsg["error"])
		fmt.Printf("\t* Value: %v\n", restMsg["value"])
	case "get":
		if len(karArgs) < 1+1 {
			fmt.Println("Error: too few arguments.")
			fmt.Println(commandUsage["kar get"])
			return
		}
		msg := getArgs(karArgs,
			[]string { "subsystem", "actorType", "actorId" },
			map[string]string {}, map[string]string{},
			1,
		)
		msg["command"] = "kar get"

		msgBytes, _ := json.Marshal(msg)
		err := sendDebugger(conn, msgBytes)
		if err != nil {
			fmt.Printf("Error sending message to debugger: %v\n", err)
			return
		}

		/* response: directly from sidecar */
		responseBytes, err := recvDebugger(connReader)
		if err != nil {
			fmt.Printf("Error receiving response from debugger: %v\n", err)
			return
		}

		getMsg := map[string]interface{} {}
		json.Unmarshal(responseBytes, &getMsg)

		if getMsg["command"] == "error" {
			fmt.Printf("Error: %v\nSee below for information on how to use this command:\n\n", getMsg["error"])
			fmt.Printf(commandUsage["kar get"])
			return
		}

		if getMsg["subsystem"] == "sidecars" {
			sidecarStr, _ := getMsg["sidecars"].(string)
			karTopology := make(map[string]sidecarData_t)
			json.Unmarshal([]byte(sidecarStr), &karTopology)
			var sb strings.Builder
			fmt.Fprint(&sb, "\nSidecar : host : port \n : Actors\n : Services")
			for sidecar, sidecarInfo := range karTopology {
				fmt.Fprintf(&sb, "\n%v", sidecar)
				fmt.Fprintf(&sb, " : %v : %v",
					sidecarInfo.Host, sidecarInfo.Port)
					fmt.Fprintf(&sb, "\n : %v\n : %v", sidecarInfo.Actors, sidecarInfo.Services)
			}
			fmt.Println(sb.String())

		} else if getMsg["subsystem"] == "actors" {
			stateStr, ok := getMsg["state"].(string)
			if ok {
				state := map[string]interface{} {}
				err = json.Unmarshal([]byte(stateStr), &state)
				fmt.Println(err)
				output, _ := json.MarshalIndent(state, "", "  ")
				fmt.Println(string(output))
			} else {
				actorsStr, _ := getMsg["actors"].(string)
				actorInfo := map[string][]string {}
				json.Unmarshal([]byte(actorsStr), &actorInfo)
				var str strings.Builder
				for actorType, actorIDs := range actorInfo {
					sort.Strings(actorIDs)
					fmt.Fprintf(&str, "%v: [\n", actorType)
					for _, actorID := range actorIDs {
						fmt.Fprintf(&str, "    %v\n", actorID)
					}
					fmt.Fprintf(&str, "]\n")
				}
				fmt.Println(str.String())
			}
		}
	default:
		fmt.Println(commandUsage["kar"])
	}
}

func processClient() {
	// variable used in displaying help messages
	help := false

	// first, process help
	argsMap := getArgs(os.Args, []string{"cmdName"}, map[string]string {
		"-v": "verbose",
	}, map[string]string {
		"-h": "help",
		"-help": "",
	}, 1)
	cmdName := argsMap["cmdName"]
	// deal with help
	if _, ok := argsMap["help"]; ok && cmdName != "kar"{
		fmt.Println(commandUsage[cmdName])
		return
	}

	// first, connect to the debugger
	// TODO: use flag package
	hostPortMap := getArgs(os.Args, []string{},
		map[string]string {
			"-host": "",
			"-port": "",
		}, map[string]string{}, 1,
	)

	debuggerHost := hostPortMap["host"]
	debuggerPort := hostPortMap["port"]

	if debuggerHost == "" {
		debuggerHost = os.Getenv("KAR_DEBUGGER_HOST")
	}
	if debuggerPort == "" {
		debuggerPort = os.Getenv("KAR_DEBUGGER_PORT")
	}

	if debuggerHost == "" {
		debuggerHost = "localhost"
	}
	if debuggerPort == "" {
		debuggerPort = "5364"
	}

	clientConn, err := net.Dial("tcp", debuggerHost + ":" + debuggerPort)
	if err != nil {
		fmt.Printf("Error connecting to debugger server: ")
		fmt.Printf("\t%v:\n\n", err)

		fmt.Println("Have you started a debugger server on this host?")
		fmt.Printf("If not, then in another terminal, run \"%s server karHost karPort\"\n", os.Args[0])
		fmt.Printf("where karHost is the hostname of a KAR sidecar in your cluster\n")
		fmt.Printf("and karPort is the port.\n")
		fmt.Printf("Then, try running this command again.\n")
		return
	}

	// connected to debugger
	
	connReader := bufio.NewReader(clientConn)

	switch cmdName {
	case "unpause":
		msg := getArgs(os.Args,
			[]string { "actorType", "actorId" },
			map[string]string {"-node": ""}, map[string]string{},
			2,
		)

		if (msg["actorType"] == "" && msg["actorId"] != "") || (msg["actorType"] != "" && msg["actorId"] == "") {
			fmt.Println("If one of actorType or actorId is given, then both must be given.")
			fmt.Println("Usage details:")
			fmt.Println(commandUsage["unpause"])
			return
		}

		msg["command"] = "unpause"

		msgBytes, _ := json.Marshal(msg)
		err = sendDebugger(clientConn, msgBytes)
		if err != nil {
			fmt.Printf("Error sending message to debugger: %v\n", err)
			return
		}

		// assume no need for response
	case "b":
		msg := getArgs(os.Args,
			[]string { "actorType", "path" },
			map[string]string {
				"-node": "",
				"-location": "isRequest",
				"-actorId": "",
				"-type": "breakpointType",
				"-conds": "",
				"-respConds": "",
			}, map[string]string{},
			2,
		)
		_, pathOk := msg["path"];
		_, actorTypeOk := msg["actorType"];
		if !(pathOk && actorTypeOk) {
			fmt.Println("Missing a required argument.")
			fmt.Println("Usage details:")
			fmt.Println(commandUsage["b"])
			return
		}
		msg["path"] = "/"+msg["path"]
		msg["command"] = "setBreakpoint"

		msgBytes, _ := json.Marshal(msg)
		err = sendDebugger(clientConn, msgBytes)
		if err != nil {
			fmt.Printf("Error sending message to debugger: %v\n", err)
			return
		}

		// response: {"breakpointId": "abc-def"}
		responseBytes, err := recvDebugger(connReader)
		if err != nil {
			fmt.Printf("Error receiving response from debugger: %v\n", err)
			return
		}
		response := map[string]string {}
		err = json.Unmarshal(responseBytes, &response)
		if err != nil {
			fmt.Printf("Error unmarshalling response: %v\n", err)
			return
		}
		fmt.Printf("Breakpoint %v set.\n", response["breakpointId"])
	case "d":
		msg := getArgs(os.Args,
			[]string { "breakpointId" },
			map[string]string {}, map[string]string{},
			2,
		)
		if msg["breakpointId"] == "" {
			fmt.Println("Missing a required argument.")
			fmt.Println("Usage details:")
			fmt.Println(commandUsage["d"])
			return
		}
		msg["command"] = "unsetBreakpoint"

		msgBytes, _ := json.Marshal(msg)
		err = sendDebugger(clientConn, msgBytes)
		if err != nil {
			fmt.Printf("Error sending message to debugger: %v\n", err)
			return
		}
		// assume no need for response
	case "vd":
		// view deadlocks

		// TODO: really refactor
		msg := getArgs([]string{},
			[]string {  },
			map[string]string {}, map[string]string{},
			0,
		)
		msg["command"] = "viewDeadlocks"
		msgBytes, _ := json.Marshal(msg)
		err = sendDebugger(clientConn, msgBytes)
		if err != nil {
			fmt.Printf("Error sending message to debugger: %v\n", err)
			return
		}

		responseBytes, err := recvDebugger(connReader)
		if err != nil {
			fmt.Printf("Error receiving response from debugger: %v\n", err)
			return
		}

		var response []edge_t
		json.Unmarshal(responseBytes, &response)
		if len(response) == 0 {
			fmt.Println("No deadlocks found.")
		} else {
			fmt.Println("Deadlock detected!")
			fmt.Println("Cycle information:")

			for i, edge := range response[1:] {
				fmt.Printf("* ")
				if i == len(response)-2 {
					fmt.Printf("But ")
				}
				fmt.Printf("%v %v is waiting on %v %v", edge.SrcActor.ActorType, edge.SrcActor.ActorId, edge.DstActor.ActorType, edge.DstActor.ActorId)
				if i == len(response)-2 {
					fmt.Printf("!")
				} else { fmt.Printf(".") }
				fmt.Printf("\n")
				rv, _ := unpackRequestValue(edge.Req.RequestValue)
				fmt.Printf("\t* Waiting on method %v.%v()\n", edge.DstActor.ActorId, rv["path"].(string)[1:])
				prv, _ := unpackRequestValue(edge.ParentValue)
				fmt.Printf("\t* Method is being called from: %v.%v()\n", edge.SrcActor.ActorId, prv["path"].(string)[1:])
			}
		}
		//pretty, _ := json.MarshalIndent(response, "", " ")
		//fmt.Printf("%s\n", pretty)
	case "vb":
		//view breakpoints
		//vb [breakpointId]
		msg := getArgs(os.Args,
			[]string { "breakpointId" },
			map[string]string {"-format": ""}, map[string]string{},
			2,
		)
		msg["command"] = "viewBreakpoint"
		msgBytes, _ := json.Marshal(msg)
		err = sendDebugger(clientConn, msgBytes)
		if err != nil {
			fmt.Printf("Error sending message to debugger: %v\n", err)
			return
		}

		/* response:
			either {"breakpoint": breakpoint_t}
			or {"breakpoints": []breakpoint_t}
		*/

		responseBytes, err := recvDebugger(connReader)
		if err != nil {
			fmt.Printf("Error receiving response from debugger: %v\n", err)
			return
		}

		if _, ok := msg["breakpointId"]; ok {
			// only expecting one breakpoint
			response := map[string]breakpoint_t {}
			err := json.Unmarshal(responseBytes, &response)
			if err != nil {
				fmt.Printf("Error unmarshalling response: %v\n", err)
				return
			}
			bk, ok := response["breakpoint"]
			if !ok {
				fmt.Printf("Error getting breakpoint data: no breakpoint data provided\n")
				return
			}
			if msg["format"] == "json" {
				pretty, _ := json.MarshalIndent(bk, "", " ")	
				fmt.Printf("%s\n", pretty)
			} else {
				printBreakpoint(bk)
			}
		} else {
			// expecting list of breakpoints
			response := map[string][]breakpoint_t {}
			err := json.Unmarshal(responseBytes, &response)
			if err != nil {
				fmt.Printf("Error unmarshalling response: %v\n", err)
				return
			}
			bks, ok := response["breakpoints"]
			if !ok {
				fmt.Printf("Error getting breakpoint data: no breakpoint data provided\n")
				return
			}

			if msg["format"] == "json" {
				pretty, _ := json.MarshalIndent(bks, "", " ")	
				fmt.Printf("%s\n", pretty)
			} else {
				if len(bks) == 0 {
					fmt.Printf("No breakpoints are currently set.")
				} else {
					for _, bk := range(bks) {
						printBreakpoint(bk)
					}
				}
			}
		}
	case "vkr":
		// view Kafka request details
		msg := getArgs(os.Args,
			[]string {"requestId"},
			map[string]string {"-format": "", "-ind": "", "-conds": ""}, map[string]string{},
			2,
		)

		msg["command"] = "vkr"
		msgBytes, _ := json.Marshal(msg)
		err = sendDebugger(clientConn, msgBytes)
		if err != nil {
			fmt.Printf("Error sending message to debugger: %v\n", err)
			return
		}
		// expecting map of reqMsg_t
		response := map[string]kafkaReq_t {}
		//response := []reqMsg_t {}
		responseBytes, err := recvDebugger(connReader)
		if err != nil {
			fmt.Printf("Error receiving from debugger: %v\n", err)
		}

		// TODO: add error checking
		err = json.Unmarshal(responseBytes, &response)
		if err != nil {
			fmt.Printf("Error unmarshalling response: %v\n", err)
			fmt.Printf("Response: %v\n", string(responseBytes))
			return
		}

		for _, reqMsg := range response {
			//fmt.Printf("* %v\n", reqMsg)
			printKafkaReq(reqMsg)
		}

		//viewRequestDetails(req, msg["requestId"])
	case "vrd":
		// view request details
		msg := getArgs(os.Args,
			[]string {"requestId"},
			map[string]string {"-format": "", "-ind": ""}, map[string]string{},
			2,
		)

		msg["command"] = "viewRequest"
		msgBytes, _ := json.Marshal(msg)
		err = sendDebugger(clientConn, msgBytes)
		if err != nil {
			fmt.Printf("Error sending message to debugger: %v\n", err)
			return
		}
		// expecting list of actors
		response := map[string]actorSentInfo_t {}
		responseBytes, err := recvDebugger(connReader)
		if err != nil {
			fmt.Printf("Error receiving from debugger: %v\n", err)
		}

		// TODO: add error checking
		err = json.Unmarshal(responseBytes, &response)
		if err != nil {
			fmt.Printf("Error unmarshalling response: %v\n", err)
			fmt.Printf("Response: %v\n", string(responseBytes))
			return
		}

		req, ok := response["request"]
		if !ok {
			fmt.Println("Request not found.")
			return
		}
		viewRequestDetails(req, msg["requestId"])
	case "vpa":
		// view paused actors
		msg := getArgs(os.Args,
			[]string {"actorType", "actorId"},
			map[string]string {"-format": "", "-ind": "",
				"-actorType": "",
				"-actorId": "",
				"-requestId": "",
				"-method": "",
				"-requestType": "",
				"-isResponse": "",
				"-breakpointId": "",
				"-nodeId": "",
			}, map[string]string{},
			2,
		)

		/*if (msg["actorType"] == "" && msg["actorId"] != "") ||
			(msg["actorType"] != "" && msg["actorId"] == "") {
			fmt.Printf("If actorType is given, then actorId must also be given.\n")

			fmt.Println("Usage details:")
			fmt.Println(commandUsage["vpa"])
			return
		}*/

		msg["command"] = "viewPausedActor"
		msgBytes, _ := json.Marshal(msg)
		err = sendDebugger(clientConn, msgBytes)
		if err != nil {
			fmt.Printf("Error sending message to debugger: %v\n", err)
			return
		}

		/* response:
			either {"actor": listPauseInfo_t }
			or {"actors": []listPauseInfo_t }
		*/

		/*if _, ok := msg["actorId"]; ok {
			// only expecting one actor
			response := map[string]listPauseInfo_t {}
			responseBytes, _ := recvDebugger(connReader)
			// TODO: add error checking
			err := json.Unmarshal(responseBytes, &response)
			if err != nil {
				fmt.Printf("Error unmarshalling response: %v\n", err)
				fmt.Printf("Response: %v\n", string(responseBytes))
				return
			}
			actor, ok := response["actor"]
			if msg["format"] == "json" {
				actorMap := map[string]interface{} {}
				actorBytes, _ := json.Marshal(actor)
				json.Unmarshal(actorBytes, &actorMap)
				actorMap["requestValue"], _ = unpackRequestValue(actor.RequestValue)
				actorMap["responseValue"], _ = unpackResponseValue(actor.ResponseValue)

				pretty, _ := json.MarshalIndent(actorMap, "", " ")	
				fmt.Printf("%s\n", pretty)
			} else {
				// assume that if no actor data is provided, it means actor is not recognized as paused
				if !ok {
					fmt.Printf("This actor is not currently paused.\n")
					return
				}
				printPausedActorFull(actor)
			}
		} else {*/
			// expecting list of actors
			response := map[string][]listPauseInfo_t {}
			responseBytes, err := recvDebugger(connReader)
			if err != nil {
				fmt.Printf("Error receiving from debugger: %v\n", err)
			}

			// TODO: add error checking
			err = json.Unmarshal(responseBytes, &response)
			if err != nil {
				fmt.Printf("Error unmarshalling response: %v\n", err)
				fmt.Printf("Response: %v\n", string(responseBytes))
				return
			}
			actors, ok := response["actors"]
			if msg["format"] == "json" {
				actorList := []map[string]interface{} {}
				for _, actor := range actors {
					actorMap := map[string]interface{} {}
					actorBytes, _ := json.Marshal(actor)
					json.Unmarshal(actorBytes, &actorMap)
					actorMap["requestValue"], _ = unpackRequestValue(actor.RequestValue)
					actorMap["responseValue"], _ = unpackResponseValue(actor.ResponseValue)

					actorList = append(actorList, actorMap)
				}

				pretty, _ := json.MarshalIndent(actorList, "", " ")	
				fmt.Printf("%s\n", pretty)
			} else {
				if !ok {
					fmt.Printf("Error: no actor data provided.\n")
					return
				}
				for _, actor := range actors {
					printPausedActorFull(actor)
				}
			}
		//}
	case "kar":
		karArgs := getArgsList(os.Args, map[string]string {
				"-v": "verbose",
			}, map[string]string {
				"-h": "help",
				"-help": "",
			}, 2)
		processClientKar(karArgs, clientConn, connReader)
	case "step":
		commandId := uuid.New().String()
		msg := getArgs(os.Args,
			[]string { "actorType", "actorId" },
			map[string]string {}, map[string]string{},
			2,
		)
		actorId, actorIdOk := msg["actorId"];
		actorType, actorTypeOk := msg["actorType"];
		if !(actorIdOk && actorTypeOk) {
			fmt.Println("Missing a required argument.")
			fmt.Println("Usage details:")
			fmt.Println(commandUsage["step"])
			return
		}

		// first, see if our actor is paused
		pauseMsg := map[string]string {
			"actorType": actorType,
			"actorId": actorId,
			"command": "viewPausedActor",
			"commandId": commandId,
			"keepAlive": "true",
			"ind": "true",
		}
		pauseMsgBytes, _ := json.Marshal(pauseMsg)
		err = sendDebugger(clientConn, pauseMsgBytes)
		if err != nil {
			fmt.Printf("Error sending pause message to debugger: %v\n", err)
			return
		}

		pauseResponse := map[string][]listPauseInfo_t {}
		pauseResponseBytes, _ := recvDebugger(connReader)

		err := json.Unmarshal(pauseResponseBytes, &pauseResponse)
		if err != nil {
			fmt.Printf("Error unmarshalling response: %v\n", err)
			fmt.Printf("Response: %v\n", string(pauseResponseBytes))
			return
		}
		pauseList, ok := pauseResponse["actors"]

		if len(pauseList) == 0 || !ok {
			fmt.Println("Cannot step: actor is not paused.")
			return
		}

		pauseInfo := pauseList[0]

		// see if our actor is paused on a response and has no parent
		// if so, then we cannot step (nothing to step to)
		if pauseInfo.IsResponse == "response" && !pauseInfo.CanStep {
			fmt.Println("Cannot step: actor is paused on a response and has no parent.")
			return
		}

		// set a flow breakpoint
		// TODO: keep old code, split into step and stepi commands

		bkId := fmt.Sprintf("single-step of actor %v %v",
			actorType, actorId)

		bmsg := map[string]string {
			"commandId": commandId,
			"flowId": pauseInfo.FlowId,
			"actorId": actorId,
			"actorType": actorType,
			"command": "setBreakpoint",
			"keepAlive": "true",
			"deleteOnHit": "true",
			"breakpointId": bkId,
		}

		/*
		// next, see if our actor is paused on a response
		if pauseInfo.IsResponse == "response" {
			fmt.Println("Cannot step: actor is paused on a response, not a request.")
			return
		}

		// now, set a breakpoint on response
		// TODO: this could easily lead to multiple breakpoints being set on the same condition
		var reqInfo = map[string]string {}
		// TODO: error handling
		json.Unmarshal([]byte(pauseInfo.RequestValue), &reqInfo)
		bmsg := map[string]string {
			"commandId": commandId,
			"actorId": pauseInfo.ActorId,//pauseInfo.EndActorId,
			"actorType": pauseInfo.ActorType,//pauseInfo.EndActorType, 
			"command": "setBreakpoint",
			"isRequest": "response",
			"keepAlive": "true",
			"path": reqInfo["path"],
			"deleteOnHit": "true",
		}*/

		bmsgBytes, _ := json.Marshal(bmsg)
		err = sendDebugger(clientConn, bmsgBytes)
		if err != nil {
			fmt.Printf("Error sending message to debugger: %v\n", err)
			return
		}

		// response: {"breakpointId": "abc-def"}
		responseBytes, err := recvDebugger(connReader)
		if err != nil {
			fmt.Printf("Error receiving response from debugger: %v\n", err)
			return
		}
		response := map[string]string {}
		err = json.Unmarshal(responseBytes, &response)
		if err != nil {
			fmt.Printf("Error unmarshalling response: %v\n", err)
			return
		}
		fmt.Println(response)
		fmt.Printf("Single-step breakpoint %v set.\n", response["breakpointId"])
		fmt.Printf("Unpausing all actors...\n")
		upmsg := map[string]string {
			"command": "unpause",
			"commandId": commandId,
			"keepAlive": "true",
		}
		upmsgBytes, _ := json.Marshal(upmsg)
		err = sendDebugger(clientConn, upmsgBytes)
		if err != nil {
			fmt.Printf("Error sending unpause message to debugger: %v\n", err)
			return
		}
		// wait for step to be hit
		responseBytes, err = recvDebugger(connReader)
		if err != nil { return }

		var stepCompleteMsg map[string]string
		err = json.Unmarshal(responseBytes, &stepCompleteMsg)

		fmt.Println("Single-step complete.")
		fmt.Println("Current location:")
		fmt.Printf("* Actor: %v %v\n", stepCompleteMsg["actorType"], stepCompleteMsg["actorId"])
		fmt.Printf("* Request vs response: %v\n", stepCompleteMsg["isResponse"])

		// TODO: error handling
		reqInfo, _ := unpackRequestValue(stepCompleteMsg["requestValue"])
		fmt.Printf("\t\t* Request type: %s\n", reqInfo["command"].(string))
		fmt.Printf("\t\t* Request method: %s\n", reqInfo["path"].(string))
		fmt.Printf("\t\t* Request payload: %s\n", reqInfo["payload"])
	case "help":
		if len(os.Args) >= 3 {
			cmd := os.Args[2]
			fmt.Println(commandUsage[cmd])
			return
		}
		help = true
		fallthrough
	default:
		if !help {
			fmt.Println("Unsupported command.")
		}
		helpFunc()
		return
	}

	// TODO: should server close connection so that the client doesn't go into TIME_WAIT, using up ports?
	clientConn.Close()
}

func helpFunc(){
	fmt.Printf("Usage: %s [GLOBALOPTIONS] COMMAND [ARGS]\n", os.Args[0])
	fmt.Println("List of commands:")
	fmt.Printf("\t")
	for command := range commandUsage {
		fmt.Printf("\"%s\" ", command)
	}
	fmt.Printf("\n")
	fmt.Println("Type \"help COMMAND\" to learn more about a command.")
	fmt.Println(`Global options:
	-v level
		Sets verbosity level. (default 0)
		Verbosity level 1 shows information about paused actors.
		Verbosity level 2 also shows detailed information about
		JSON messages sent to the debugger clients and received
		from the KAR sidecar.
	-h, -help
		Displays help about a command.`)
}

var verbose = 0

func main(){
	if len(os.Args) == 1 {
		helpFunc()
		return
	}
	myArgs := getArgs(os.Args, []string{"cmdName"},
		map[string]string {
			"-v": "verbose",
		}, map[string]string {
			"-h": "help",
			"-help": "",
		}, 1)
	
	if _, ok := myArgs["help"]; ok {

	}
	
	verbose, _ = strconv.Atoi(myArgs["verbose"])
	cmdName := myArgs["cmdName"]

	if cmdName == "server" {
		debuggerId := uuid.New().String()

		// connect to the kar server
		serverArgs := getArgs(os.Args,
			[]string{"karHost", "karPort"},
			map[string]string{"-serverPort": ""},
			map[string]string{}, 2)

		karHost, hostOk := serverArgs["karHost"]
		karPort, portOk := serverArgs["karPort"]

		if !(hostOk && portOk) {
			karPort = os.Getenv("KAR_RUNTIME_PORT")
			if karPort != "" {
				karHost = "localhost"
			} else  {
				fmt.Printf(commandUsage["server"])
				return
			}
		}

		registerUrl = "ws://" + karHost + ":" + karPort +
			"/kar/v1/debug/register"

		headers := map[string][]string {
			"id": []string { debuggerId },
		}

		var err error
		var resp *http.Response
		conn, resp, err = websocket.DefaultDialer.Dial(registerUrl, headers)

		if err != nil {
			fmt.Printf("Error connecting to KAR sidecar: %v\n", err)
			if resp != nil {
				fmt.Printf("\tError response: %v\n", *resp)
			} else {
				fmt.Println("Make sure that you entered the correct hostname and port.")
				fmt.Println("Additionally, make sure that the KAR sidecar is running, with its port exposed.")
			}
			return
		}

		// synchronize breakpoints
		msg := map[string]string {
			"command": "listBreakpoints",
		}
		msgBytes, _ := json.Marshal(msg)
		err = send(string(msgBytes))
		if err != nil {
			fmt.Printf("Error getting breakpoints from KAR sidecar: %v\n", err)
			fmt.Printf("\tError response: %v\n", *resp)
			return
		}

		// synchronize paused actors
		msg = map[string]string {
			"command": "listPausedActors",
		}
		msgBytes, _ = json.Marshal(msg)
		err = send(string(msgBytes))
		if err != nil {
			fmt.Printf("Error getting paused actors from KAR sidecar: %v\n", err)
			fmt.Printf("\tError response: %v\n", *resp)
			return
		}

		go listenSidecar()

		// listen as a debugger server

		serverPort := serverArgs["serverPort"]
		/*if serverPort == "" {
			serverPort = os.Getenv("KAR_APP_PORT")
		}//os.Getenv("KAR_DEBUG_SERVER_PORT")*/
		if serverPort == "" { serverPort = "5364" }
		ln, err := net.Listen("tcp", ":"+serverPort)
		if err != nil {
			fmt.Printf("Error listening as debugger server: %v", err)
			return
		}

		fmt.Printf("Debugger server connected to sidecar.\n")
		fmt.Printf("Listening on port %s.\n", serverPort)

		//accept connections
		
		for {
			conn, err := ln.Accept()
			if err != nil {
				conn.Close()
				continue
			}
			go serveDebugger(conn)
		}
	
	} else {
		processClient()
	}
}
