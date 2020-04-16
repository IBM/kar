//go:generate swagger generate spec

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
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
	finished      = make(chan struct{})                      // wait for http server to complete shutdown
)

// swagger:route POST /service/{service}/tell/{path} services idTellService
//
// tell: Asynchronously invoke a service.
//
// Tell asynchronously executes a `POST` to the `path` endpoint of `service` passing
// through the optional JSON payload it received. A `200` response indicates that
// the request has been accepted by the runtime and will eventually be delivered to
// the targeted service endpoint.
//
//     Consumes: application/json
//     Schemes: http, https
//     Responses:
//       200: response200
//       500: response500
//       503: response503
//

// swagger:route POST /actor/{actorType}/{actorId}/tell/{path} actors idTellActor
//
// tell: Asynchronosuly invoke an actor.
//
// Actor-tell asynchronously executes a `POST` to the `path` endpoint of the
// actor instance indicated by `actorType` and `actorId` passing through
// the optional JSON payload it received.  A `200` response indicates that
// the request has been accepted by the runtime and will eventually be delivered to
// the targeted actor method.
//
//     Consumes: application/json
//     Schemes: http, https
//     Responses:
//       200: response200
//       500: response500
//       503: response503
//
func tell(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var err error
	if ps.ByName("service") != "" {
		err = runtime.TellService(ctx, ps.ByName("service"), ps.ByName("path"), runtime.ReadAll(r.Body), r.Header.Get("Content-Type"))
	} else {
		err = runtime.TellActor(ctx, runtime.Actor{Type: ps.ByName("type"), ID: ps.ByName("id")}, ps.ByName("path"), runtime.ReadAll(r.Body), r.Header.Get("Content-Type"))
	}
	if err != nil {
		if ctx.Err() != nil {
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		} else {
			http.Error(w, fmt.Sprintf("failed to send message: %v", err), http.StatusInternalServerError)
		}
	} else {
		fmt.Fprint(w, "OK")
	}
}

// swagger:route POST /system/broadcast/{path} system idBroadcast
//
// broadcast: send message to all KAR runtimes.
//
// The broadcast route causes a `POST` of `path` to be delivered to all
// KAR runtime processes that are currently part of the application.
// A `200` response indicates that the request to send the broadcast
// has been accepted and the POST will eventually be delivered to all sidecars.
//
//     Consumes: application/json
//     Produces: application/json
//     Schemes: http, https
//     Responses:
//       200: response200
//
func broadcast(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	runtime.Broadcast(ctx, ps.ByName("path"), runtime.ReadAll(r.Body), r.Header.Get("Content-Type"))
	fmt.Fprint(w, "OK")
}

// swagger:route POST /service/{service}/call/{path} services idCallService
//
// call: Synchronously invoke a service.
//
// Call synchronously executes a `POST` to the `path` endpoint of `service` passing
// through an optional JSON payload to the service and responding with the
// result returned by the service.
//
//     Consumes: application/json
//     Produces: application/json
//     Schemes: http, https
//     Responses:
//       200: callPath200Response
//       500: response500
//       503: response503
//

// swagger:route POST /actor/{actorType}/{actorId}/call/{path} actors idCallActor
//
// call: Synchronously invoke an actor.
//
// Call synchronously executes a `POST` to the `path` endpoint of the
// actor instance indicated by `actorType` and `actorId` passing
// through an optional JSON payload to the actor and responding with the
// result returned by the actor method.
//
//     Consumes: application/json
//     Produces: application/json
//     Schemes: http, https
//     Responses:
//       200: callPath200Response
//       500: response500
//       503: response503
//
func call(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var reply *runtime.Reply
	var err error
	if ps.ByName("service") != "" {
		reply, err = runtime.CallService(ctx, ps.ByName("service"), ps.ByName("path"), runtime.ReadAll(r.Body), r.Header.Get("Content-Type"), r.Header.Get("Accept"))
	} else {
		session := r.FormValue("session")
		reply, err = runtime.CallActor(ctx, runtime.Actor{Type: ps.ByName("type"), ID: ps.ByName("id")}, ps.ByName("path"), runtime.ReadAll(r.Body), r.Header.Get("Content-Type"), r.Header.Get("Accept"), session)
	}
	if err != nil {
		if ctx.Err() != nil {
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

// swagger:route GET /actor/{actorType}/{actorId}/migrate actors idActorMigrate
//
// migrate: Request the migration of an actor
//
// TODO: Operation detailed description
//
//     Schemes: http, https
//     Responses:
//       200: response200
//
func migrate(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	runtime.Migrate(ctx, runtime.Actor{Type: ps.ByName("type"), ID: ps.ByName("id")})
	fmt.Fprint(w, "OK")
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

// swagger:route DELETE /actor/{actorType}/{actorId}/reminder actors idCancelReminder
//
// reminder: Cancel all matching reminders.
//
// This operation cancels reminders for the actor specified in the path.
// If a reminder id is provided as a parameter, only the reminder whose id
// matches that id will be cancelled. If no id is provided, all
// of the specified actor's reminders will be cancelled.  The number of reminders
// actually cancelled is returned as the result of the operation.
//
//     Consumes: application/json
//     Produces: application/json
//     Schemes: http, https
//     Responses:
//       200: cancelReminder200Response
//       500: response500
//       503: response503
//

// swagger:route GET /actor/{actorType}/{actorId}/reminder actors idGetReminder
//
// reminder: Get all matching reminders.
//
// This operatation returns all reminders for the actor(s) specified in the path.
// If a reminder id is provided as a parameter, only reminders that
// have that id will be returned.
//
//     Consumes: application/json
//     Produces: application/json
//     Schemes: http, https
//     Responses:
//       200: getReminder200Response
//       500: response500
//       503: response503
//

// swagger:route POST /actor/{actorType}/{actorId}/reminder actors idScheduleReminder
//
// reminder: Schedule a reminder.
//
// This operatation schedules a reminder for the actor specified in the path.
// Consistient with the expected semantics of a `POST` operation, if there is
// already a reminder for the actor with the same reminderId, that
// existing reminder's schedule will be updated based on the request body.
// The method will not return until after the reminder is scheduled.
//
//     Consumes: application/json
//     Produces: application/json
//     Schemes: http, https
//     Responses:
//       200: response200
//       500: response500
//       503: response503
//
func reminder(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var action string
	switch r.Method {
	case "GET":
		action = "get"
	case "POST":
		action = "schedule"
	case "DELETE":
		action = "cancel"
	default:
		http.Error(w, fmt.Sprintf("Unsupported method %v", r.Method), http.StatusMethodNotAllowed)
		return
	}
	reply, err := runtime.Reminders(ctx, runtime.Actor{Type: ps.ByName("type"), ID: ps.ByName("id")}, action, runtime.ReadAll(r.Body), r.Header.Get("Content-Type"), r.Header.Get("Accept"))
	if err != nil {
		if ctx.Err() != nil {
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

// swagger:route POST /actor/{actorType}/{actorId}/state/{key} actors idActorStateSet
//
// state: Store a key-value pair in an actor's state
//
// TODO: Operation detailed description
//
//     Consumes: application/json
//     Produces: application/json
//     Schemes: http, https
//
func set(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if reply, err := store.HSet(stateKey(ps.ByName("type"), ps.ByName("id")), ps.ByName("key"), runtime.ReadAll(r.Body)); err != nil {
		http.Error(w, fmt.Sprintf("HSET failed: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, reply)
	}
}

// swagger:route GET /state/{actorType}/{actorId}/state/{key} actors idActorStateGet
//
// state: Get the value associated with a key in an actor's state returning nil if not found.
//
// TODO: Operation detailed description
//
//     Consumes: application/json
//     Produces: application/json
//     Schemes: http, https
//
func get(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if reply, err := store.HGet(stateKey(ps.ByName("type"), ps.ByName("id")), ps.ByName("key")); err == store.ErrNil {
		if eoa := r.FormValue("errorOnAbsent"); eoa == "true" {
			http.Error(w, "Not Found", http.StatusNotFound)
		} else {
			fmt.Fprint(w, reply)
		}
	} else if err != nil {
		http.Error(w, fmt.Sprintf("HGET failed: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, reply)
	}
}

// swagger:route DELETE /actor/{actorType}/{actorId}/state/{key} actors idActorStateDeleteKey
//
// state: Remove a key-value pair in an actor's state.
//
// TODO: Operation detailed description
//
//     Schemes: http, https
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
// state: Get all key-value pairs in an actor's state.
//
// TODO: Operation detailed description
//
//     Consumes: application/json
//     Produces: application/json
//     Schemes: http, https
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

// swagger:route DELETE /actor/{actorType}/{actorId}/state actors idActorStateDeleteAll
//
// state: Delete all key-value pairs in an actor's state.
//
// TODO: Operation detailed description
//
//     Schemes: http, https
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

// swagger:route GET /system/kill system idKill
//
// kill: Initiate an orderly shutdown of a KAR runtime process.
//
// TODO: Operation detailed description
//
//     Schemes: http, https
//
func kill(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	logger.Info("Invoking cancel() in response to kill request")
	cancel()
	wg.Wait()
	fmt.Fprint(w, "OK")
}

// swagger:route GET /system/killall system idKillAll
//
// killall: Initiate an orderly shutdown of all of an application's KAR runtime processes.
//
// TODO: Operation detailed description
//
//     Schemes: http, https
//
func killall(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	runtime.KillAll(ctx)
	fmt.Fprint(w, "OK")
	cancel()
}

// swagger:route GET /system/health system health
//
// health: Health-check endpoint of a KAR runtime process.
//
// TODO: Operation detailed description
//
//     Schemes: http, https
//
func health(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprint(w, "OK")
}

// swagger:route POST /event/{topic}/publish events eventPublishId
//
// publish: Publish an event to a topic.
//
// TODO: Operation detailed description
//
//     Schemes: http, https
//
func publish(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	reply, err := pubsub.Publish(ps.ByName("topic"), runtime.ReadAll(r.Body))
	if err != nil {
		http.Error(w, fmt.Sprintf("publish error: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, reply)
	}
}

// swagger:route POST /event/{topic}/subscribe events eventSubscribeId
//
// subscribe: Subscribe to a topic.
//
// TODO: Operation detailed description
//
//     Schemes: http, https
//
func subscribe(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	reply, err := runtime.Subscribe(ctx, ps.ByName("topic"), runtime.ReadAll(r.Body))
	if err != nil {
		http.Error(w, fmt.Sprintf("subscribe error: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, reply)
	}
}

// swagger:route POST /event/{topic}/unsubscribe events eventUnsubscribeId
//
// unsubscribe: Unsubscribe from a topic.
//
// TODO: Operation detailed description
//
//     Schemes: http, https
//
func unsubscribe(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	reply, err := runtime.Unsubscribe(ctx, ps.ByName("topic"), runtime.ReadAll(r.Body))
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
	router.POST(base+"/service/:service/tell/*path", tell)

	//actor invocation
	router.POST(base+"/actor/:type/:id/call/*path", call)
	router.POST(base+"/actor/:type/:id/tell/*path", tell)
	//
	router.GET(base+"/actor/:type/:id/migrate", migrate)
	//
	router.GET(base+"/actor/:type/:id/reminder", reminder)
	router.POST(base+"/actor/:type/:id/reminder", reminder)
	router.DELETE(base+"/actor/:type/:id/reminder", reminder)
	//
	router.GET(base+"/actor/:type/:id/state/:key", get)
	router.POST(base+"/actor/:type/:id/state/:key", set)
	router.DELETE(base+"/actor/:type/:id/state/:key", del)
	router.GET(base+"/actor/:type/:id/state", getAll)
	router.DELETE(base+"/actor/:type/:id/state", delAll)

	// kar system methods
	router.POST(base+"/system/broadcast/*path", broadcast)
	router.GET(base+"/system/health", health)
	router.GET(base+"/system/kill", kill)
	router.GET(base+"/system/killall", killall)

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

	go func() {
		defer close(finished)
		<-ctx.Done() // wait
		if err := srv.Shutdown(context.Background()); err != nil {
			logger.Fatal("failed to shutdown HTTP server: %v", err)
		}
	}()
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

	channel := pubsub.Dial(ctx)
	defer pubsub.Close()

	store.Dial()
	defer store.Close()

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

	port1 := fmt.Sprintf("KAR_PORT=%d", listener.Addr().(*net.TCPAddr).Port)
	port2 := fmt.Sprintf("KAR_APP_PORT=%d", config.ServicePort)
	logger.Info("%s %s", port1, port2)

	args := flag.Args()

	if len(args) > 0 {
		exitCode = runtime.Run(ctx9, args, append(os.Environ(), port1, port2))
		cancel()
	}

	wg.Wait()

	<-finished

	logger.Warning("exiting...")
}
