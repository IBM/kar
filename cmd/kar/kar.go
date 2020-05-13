//go:generate swagger generate spec

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/textproto"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/julienschmidt/httprouter"
	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/internal/pubsub"
	"github.ibm.com/solsa/kar.git/internal/runtime"
	"github.ibm.com/solsa/kar.git/internal/store"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

var (
	// termination
	ctx9, cancel9 = context.WithCancel(context.Background()) // preemptive: kill subprocess
	ctx, cancel   = context.WithCancel(ctx9)                 // cooperative: wait for subprocess
	wg            = &sync.WaitGroup{}                        // wait for kafka consumer and http server to stop processing requests
)

func tell(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var err error
	if ps.ByName("service") != "" {
		err = runtime.TellService(ctx, ps.ByName("service"), ps.ByName("path"), runtime.ReadAll(r), r.Header.Get("Content-Type"))
	} else {
		err = runtime.TellActor(ctx, runtime.Actor{Type: ps.ByName("type"), ID: ps.ByName("id")}, ps.ByName("path"), runtime.ReadAll(r), r.Header.Get("Content-Type"))
	}
	if err != nil {
		if err == ctx.Err() {
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		} else {
			http.Error(w, fmt.Sprintf("failed to send message: %v", err), http.StatusInternalServerError)
		}
	} else {
		fmt.Fprint(w, "OK")
	}
}

// swagger:route POST /system/broadcast/{path} system idSystemBroadcast
//
// broadcast
//
// ### Asynchronously broadcast a message to the KAR runtime
//
// Broadcast asynchronously executes a `POST` on the `path` endpoint
// of all other KAR runtimes that are currently part of the application.
// The runtime initiating the broadcast is not included as a recipient.
// A `200` response indicates that the request to send the broadcast
// has been accepted and the POST will eventually be delivered to all targeted
// runtime processes.
//
// This route is Deprecated and slated for removal from the API.
//
//     Deprecated: true
//     Consumes:
//     - application/json
//     Produces:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200
//
func broadcast(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	runtime.Broadcast(ctx, ps.ByName("path"), runtime.ReadAll(r), r.Header.Get("Content-Type"))
	fmt.Fprint(w, "OK")
}

// swagger:route POST /service/{service}/call/{path} services idServiceCall
//
// call
//
// ### Invoke a service endpoint
//
// Call executes a `POST` to the `path` endpoint of `service`.
// The JSON request body is passed through to the target endpoint.
// The result of the call is the result of invoking the target service endpoint
// unless the `async` pragma header is specified.
//
//     Consumes:
//     - application/json
//     Produces:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200CallResult
//       500: response500
//       503: response503
//       default: responseGenericEndpointError
//

// swagger:route POST /actor/{actorType}/{actorId}/call/{path} actors idActorCall
//
// call
//
// ### Invoke an actor method
//
// Call executes a `POST` to the `path` endpoint of the
// actor instance indicated by `actorType` and `actorId`.
// The JSON request body is passed through to the target endpoint.
// The result of the call is the result of invoking the target actor method
// unless the `async` pragma header is specified.
//
//     Consumes:
//     - application/json
//     Produces:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200CallResult
//       500: response500
//       503: response503
//       default: responseGenericEndpointError
//
func call(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	for _, pragma := range r.Header[textproto.CanonicalMIMEHeaderKey("Pragma")] {
		if strings.ToLower(pragma) == "async" {
			tell(w, r, ps)
			return
		}
	}
	var reply *runtime.Reply
	var err error
	if ps.ByName("service") != "" {
		reply, err = runtime.CallService(ctx, ps.ByName("service"), ps.ByName("path"), runtime.ReadAll(r), r.Header.Get("Content-Type"), r.Header.Get("Accept"))
	} else {
		session := r.FormValue("session")
		reply, err = runtime.CallActor(ctx, runtime.Actor{Type: ps.ByName("type"), ID: ps.ByName("id")}, ps.ByName("path"), runtime.ReadAll(r), r.Header.Get("Content-Type"), r.Header.Get("Accept"), session)
	}
	if err != nil {
		if err == ctx.Err() {
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		} else {
			http.Error(w, fmt.Sprintf("failed to send message: %v", err), http.StatusInternalServerError)
		}
	} else {
		w.Header().Add("Content-Type", reply.ContentType)
		w.WriteHeader(reply.StatusCode)
		fmt.Fprint(w, reply.Payload)
	}
}

// swagger:route POST /actor/{actorType}/{actorId}/migrate actors idActorMigrate
//
// migrate
//
// ### Initiate an actor migration
//
// This operation is primarily intended to be used by the KAR actor runtime.
// When delivered to the runtime currently hosting the designated actor instance,
// it causes the actor to be passivated and the binding of the actor instance to
// that runtime to be removed from the KAR actor placement service. When next
// activated, the actor instance may be hosted by a different instance of the
// application process.
//
//     Schemes: http
//     Responses:
//       200: response200
//       500: response500
//       503: response503
//
func migrate(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	err := runtime.Migrate(ctx, runtime.Actor{Type: ps.ByName("type"), ID: ps.ByName("id")}, "")
	if err != nil {
		if err == ctx.Err() {
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		} else {
			http.Error(w, fmt.Sprintf("failed to migrate actor: %v", err), http.StatusInternalServerError)
		}
	} else {
		fmt.Fprint(w, "OK")
	}
}

// swagger:route DELETE /actor/{actorType}/{actorId}/reminders actors idActorReminderCancelAll
//
// reminders
//
// ### Cancel all reminders
//
// This operation cancels all reminders for the actor instance specified in the path.
// The number of reminders cancelled is returned as the result of the operation.
//
//     Consumes:
//     - application/json
//     Produces:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200ReminderCancelAllResult
//       500: response500
//       503: response503
//

// swagger:route DELETE /actor/{actorType}/{actorId}/reminders/{reminderId} actors idActorReminderCancel
//
// reminders/id
//
// ### Cancel a reminder
//
// This operation cancels the reminder for the actor instance specified in the path.
// If the reminder is successfully cancelled a `200` response with a body of `1` will be returned.
// If the reminder is not found, a `404` response will be returned unless
// the boolean query parameter `nilOnAbsent` is set to `true`. If `nilOnAbsent`
// is sent to true the `404` response will instead be a `200` with a body containing `0`.
//
//     Consumes:
//     - application/json
//     Produces:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200ReminderCancelResult
//       404: response404
//       500: response500
//       503: response503
//

// swagger:route GET /actor/{actorType}/{actorId}/reminders actors idActorReminderGetAll
//
// reminders
//
// ### Get all reminders
//
// This operation returns all reminders for the actor instance specified in the path.
//
//     Consumes:
//     - application/json
//     Produces:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200ReminderGetAllResult
//       500: response500
//       503: response503
//

// swagger:route GET /actor/{actorType}/{actorId}/reminders/{reminderId} actors idActorReminderGet
//
// reminders/id
//
// ### Get a reminder
//
// This operation returns the reminder for the actor instance specified in the path.
// If there is no reminder with the id `reminderId` a `404` response will be returned
// unless the boolean query parameter `nilOnAbsent` is set to `true`.
// If `nilOnAbsent` is true the `404` response will be replaced with
// a `200` response with a `nil` response body.
//
//     Consumes:
//     - application/json
//     Produces:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200ReminderGetResult
//       404: response404
//       500: response500
//       503: response503
//

// swagger:route POST /actor/{actorType}/{actorId}/reminders actors idActorReminderSchedule
//
// reminders
//
// ### Schedule a reminder
//
// This operation schedules a reminder for the actor instance specified in the path
// as described by the data provided in the request body.
// If there is already a reminder for the target actor instance with the same reminderId,
// that existing reminder's schedule will be updated based on the request body.
// The operation will not return until after the reminder is scheduled.
//
//     Consumes:
//     - application/json
//     Produces:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200
//       500: response500
//       503: response503
//
func reminder(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var action string
	body := ""
	noa := "false"
	switch r.Method {
	case "GET":
		action = "get"
		noa = r.FormValue("nilOnAbsent")
	case "POST":
		action = "schedule"
		body = runtime.ReadAll(r)
	case "DELETE":
		action = "cancel"
		noa = r.FormValue("nilOnAbsent")
	default:
		http.Error(w, fmt.Sprintf("Unsupported method %v", r.Method), http.StatusMethodNotAllowed)
		return
	}
	reply, err := runtime.Reminders(ctx, runtime.Actor{Type: ps.ByName("type"), ID: ps.ByName("id")}, ps.ByName("reminderId"), noa, action, body, r.Header.Get("Content-Type"), r.Header.Get("Accept"))
	if err != nil {
		if err == ctx.Err() {
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		} else {
			http.Error(w, fmt.Sprintf("failed to send message: %v", err), http.StatusInternalServerError)
		}
	} else {
		w.Header().Add("Content-Type", reply.ContentType)
		w.WriteHeader(reply.StatusCode)
		fmt.Fprint(w, reply.Payload)
	}
}

func stateKey(t, id string) string {
	return "main" + config.Separator + "state" + config.Separator + t + config.Separator + id
}

// swagger:route PUT /actor/{actorType}/{actorId}/state/{key} actors idActorStateSet
//
// state/key
//
// ### Update a single entry of an actor's state
//
// The state of the actor instance indicated by `actorType` and `actorId`
// will be updated by setting `key` to contain the JSON request body.
// The operation will not return until the state has been updated.
// The result of the operation is `1` if a new entry was created and `0` if an existing entry was updated.
//
//     Consumes:
//     - application/json
//     Produces:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200StateSetResult
//       500: response500
//
func set(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if reply, err := store.HSet(stateKey(ps.ByName("type"), ps.ByName("id")), ps.ByName("key"), runtime.ReadAll(r)); err != nil {
		http.Error(w, fmt.Sprintf("HSET failed: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, reply)
	}
}

// swagger:route GET /actor/{actorType}/{actorId}/state/{key} actors idActorStateGet
//
// state/key
//
// ### Get a single entry of an actor's state
//
// The `key` entry of the state of the actor instance indicated by `actorType` and `actorId`
// will be returned as the response body.
// If there is no entry for `key` a `404` response will be returned
// unless the boolean query parameter `nilOnAbsent` is set to `true`,
// in which case a `200` reponse with a `nil` response body will be returned.
//
//     Produces:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200StateGetResult
//       404: response404
//       500: response500
//
func get(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if reply, err := store.HGet(stateKey(ps.ByName("type"), ps.ByName("id")), ps.ByName("key")); err == store.ErrNil {
		if noa := r.FormValue("nilOnAbsent"); noa == "true" {
			fmt.Fprint(w, reply)
		} else {
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	} else if err != nil {
		http.Error(w, fmt.Sprintf("HGET failed: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, reply)
	}
}

// swagger:route DELETE /actor/{actorType}/{actorId}/state/{key} actors idActorStateDelete
//
// state/key
//
// ### Remove a single entry in an actor's state
//
// The state of the actor instance indicated by `actorType` and `actorId`
// will be updated by removing the entry for `key`.
// The operation will not return until the state has been updated.
// The result of the operation is `1` if an entry was actually removed and
// `0` if there was no entry for `key`.
//
//     Schemes: http
//     Responses:
//       200: response200StateDeleteResult
//       500: response500
//
func del(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if reply, err := store.HDel(stateKey(ps.ByName("type"), ps.ByName("id")), ps.ByName("key")); err != nil {
		http.Error(w, fmt.Sprintf("HDEL failed: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, reply)
	}
}

// swagger:route GET /actor/{actorType}/{actorId}/state actors idActorStateGetAll
//
// state
//
// ### Get an actor's state
//
// The state of the actor instance indicated by `actorType` and `actorId`
// will be returned as the response body.
//
//     Produces:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200StateGetAllResult
//       500: response500
//
func getAll(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if reply, err := store.HGetAll(stateKey(ps.ByName("type"), ps.ByName("id"))); err != nil {
		http.Error(w, fmt.Sprintf("HGETALL failed: %v", err), http.StatusInternalServerError)
	} else {
		// reply has type map[string]string
		// we unmarshal the values then marshal the map
		m := map[string]interface{}{}
		for i, s := range reply {
			var v interface{}
			json.Unmarshal([]byte(s), &v)
			m[i] = v
		}
		b, _ := json.Marshal(m)
		fmt.Fprint(w, string(b))
	}
}

// swagger:route POST /actor/{actorType}/{actorId}/state actors idActorStateSetMultiple
//
// state
//
// ### Update multiple entries of an actor's state
//
// The state of the actor instance indicated by `actorType` and `actorId`
// will be updated by atomically updated by storing all key-value pairs
// in the request body.
// The operation will not return until the state has been updated.
// The result of the operation is the number of new entires that were created.
//
//     Consumes:
//     - application/json
//     Produces:
//     - application/json
//     Schemes: http
//     Responses:
//       200: response200StateSetMultipleResult
//       400: response400
//       500: response500
//
func setMultiple(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var updates map[string]interface{}
	if err := json.Unmarshal([]byte(runtime.ReadAll(r)), &updates); err != nil {
		http.Error(w, "Request body was not a map[string, interface{}]", http.StatusBadRequest)
		return
	}
	sk := stateKey(ps.ByName("type"), ps.ByName("id"))
	m := map[string]string{}
	for i, v := range updates {
		s, err := json.Marshal(v)
		if err != nil {
			logger.Error("setMultiple: %v[%v] = %v failed due to %v", sk, i, v, err)
			http.Error(w, fmt.Sprintf("Unable to re-serialize value %v", v), http.StatusInternalServerError)
			return
		}
		m[i] = string(s)
	}
	if reply, err := store.HSetMultiple(sk, m); err != nil {
		http.Error(w, fmt.Sprintf("HSET failed: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, reply)
	}
}

// swagger:route DELETE /actor/{actorType}/{actorId}/state actors idActorStateDeleteAll
//
// state
//
// ### Remove an actor's state
//
// The state of the actor instance indicated by `actorType` and `actorId`
// will be deleted.
//
//     Schemes: http
//     Responses:
//       200: response200StateDeleteResult
//       404: response404
//       500: response500
//
func delAll(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if reply, err := store.Del(stateKey(ps.ByName("type"), ps.ByName("id"))); err == store.ErrNil {
		http.Error(w, "Not Found", http.StatusNotFound)
	} else if err != nil {
		http.Error(w, fmt.Sprintf("DEL failed: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, reply)
	}
}

// swagger:route POST /system/shutdown system idSystemShutdown
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
func shutdown(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprint(w, "OK")
	logger.Info("Invoking cancel() in response to shutdown request")
	cancel()
}

// swagger:route GET /system/health system isSystemHealth
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
func health(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprint(w, "OK")
}

// swagger:route POST /event/{topic}/publish events idEventPublish
//
// publish
//
// ### Publish an event to a topic
//
// The event provided as the request body will be published on `topic`.
// When the operation returns successfully, the event is guaranteed to
// eventually be published to the targeted topic.
//
//     Schemes: http
//     Responses:
//       200: response200
//       500: response500
//
func publish(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// FIXME: https://github.ibm.com/solsa/kar/issues/30
	//        Should return a 404 if topic doesn't exist.
	buf, _ := ioutil.ReadAll(r.Body)
	err := pubsub.Publish(ps.ByName("topic"), buf)
	if err != nil {
		http.Error(w, fmt.Sprintf("publish error: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, "OK")
	}
}

// swagger:route POST /event/{topic}/subscribe events idEventSubscribe
//
// subscribe
//
// ### Subscribe to a topic
//
// Subscribe an application endpoint to be invoked when events are delivered to
// the targeted `topic`.  The endpoint is described in the request body and
// may be either a service endpoint or an actor method.
//
//     Schemes: http
//     Consumes:
//     - application/json
//     Responses:
//       200: response200
//       500: response500
//
func subscribe(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// FIXME: https://github.ibm.com/solsa/kar/issues/31
	//        Should return a 404 if topic doesn't exist.
	reply, err := runtime.Subscribe(ctx, ps.ByName("topic"), runtime.ReadAll(r))
	if err != nil {
		http.Error(w, fmt.Sprintf("subscribe error: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, reply)
	}
}

// swagger:route POST /event/{topic}/unsubscribe events idEventUnsubscribe
//
// unsubscribe
//
// ### Unsubscribe from a topic
//
// Unsubscribe an application endpoint described by the request body from `topic`.
// The operation may return before it actually completes, but upon
// success it is guaranteed that the endpoint will eventually stop receive
// events from the topic.
//
//     Schemes: http
//     Consumes:
//     - application/json
//     Responses:
//       200: response200
//       500: response500
//
func unsubscribe(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// FIXME: https://github.ibm.com/solsa/kar/issues/31
	//        Should return a 404 if topic doesn't exist.
	reply, err := runtime.Unsubscribe(ctx, ps.ByName("topic"), runtime.ReadAll(r))
	if err != nil {
		http.Error(w, fmt.Sprintf("unsubscribe error: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, reply)
	}
}

// server implements the HTTP server
func server(listener net.Listener) {
	base := "/kar/v1"
	router := httprouter.New()
	// service invocation
	router.POST(base+"/service/:service/call/*path", call)

	//actor invocation
	router.POST(base+"/actor/:type/:id/call/*path", call)
	//
	router.POST(base+"/actor/:type/:id/migrate", migrate)
	//
	router.GET(base+"/actor/:type/:id/reminders/:reminderId", reminder)
	router.GET(base+"/actor/:type/:id/reminders", reminder)
	router.POST(base+"/actor/:type/:id/reminders", reminder)
	router.DELETE(base+"/actor/:type/:id/reminders/:reminderId", reminder)
	router.DELETE(base+"/actor/:type/:id/reminders", reminder)
	//
	router.GET(base+"/actor/:type/:id/state/:key", get)
	router.PUT(base+"/actor/:type/:id/state/:key", set)
	router.DELETE(base+"/actor/:type/:id/state/:key", del)
	router.GET(base+"/actor/:type/:id/state", getAll)
	router.POST(base+"/actor/:type/:id/state", setMultiple)
	router.DELETE(base+"/actor/:type/:id/state", delAll)

	// kar system methods
	router.POST(base+"/system/broadcast/*path", broadcast)
	router.GET(base+"/system/health", health)
	router.POST(base+"/system/shutdown", shutdown)

	// events
	router.POST(base+"/event/:topic/publish", publish)
	router.POST(base+"/event/:topic/subscribe", subscribe)
	router.POST(base+"/event/:topic/unsubscribe", unsubscribe)

	srv := http.Server{Handler: h2c.NewHandler(router, &http2.Server{MaxConcurrentStreams: 262144})}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.Serve(listener); err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed: %v", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done() // wait
		if err := srv.Shutdown(context.Background()); err != nil {
			logger.Error("failed to shutdown HTTP server: %v", err)
		}
	}()
}

// process incoming messages in parallel
func process(m pubsub.Message) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		runtime.Process(ctx, cancel, m)
	}()
}

// subscriber handles incoming messages
func subscriber(channel <-chan pubsub.Message) {
	for m := range channel {
		process(m)
	}
}

func main() {
	logger.Warning("starting...")
	exitCode := 0
	defer func() { os.Exit(exitCode) }()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signals
		logger.Info("Invoking cancel9() from signal handler")
		cancel9()
	}()

	go func() {
		<-ctx.Done()
		runtime.CloseIdleConnections()
	}()

	var listenHost string
	if config.KubernetesMode {
		listenHost = fmt.Sprintf(":%d", config.RuntimePort)
	} else {
		listenHost = fmt.Sprintf("127.0.0.1:%d", config.RuntimePort)
	}
	listener, err := net.Listen("tcp", listenHost)
	if err != nil {
		logger.Fatal("listener failed: %v", err)
	}

	if store.Dial() != nil {
		logger.Fatal("failed to connect to Redis: %v", err)
	}
	defer store.Close()

	if pubsub.Dial() != nil {
		logger.Fatal("dial failed: %v", err)
	}
	defer pubsub.Close()

	channel, err := pubsub.Join(ctx)
	if err != nil {
		logger.Fatal("join failed: %v", err)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		subscriber(channel)
	}()

	server(listener)

	wg.Add(1)
	go func() {
		defer wg.Done()
		runtime.Collect(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runtime.ProcessReminders(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runtime.ManageReminderPartitions(ctx)
	}()

	runtimePort := fmt.Sprintf("KAR_RUNTIME_PORT=%d", listener.Addr().(*net.TCPAddr).Port)
	appPort := fmt.Sprintf("KAR_APP_PORT=%d", config.AppPort)
	logger.Info("%s %s", runtimePort, appPort)

	args := flag.Args()

	if len(args) > 0 {
		exitCode = runtime.Run(ctx9, args, append(os.Environ(), runtimePort, appPort))
		cancel()
	}

	wg.Wait()

	cancel9()

	logger.Warning("exiting...")
}
