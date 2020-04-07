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

// swagger:route POST /tell/{service}/{path} services idTellService
//
// Operation summary.
//
// Operation detailed description
//
//     Consumes: application/json
//     Produces: application/json
//     Schemes: http, https
//

// swagger:route POST /actor-tell/{actorType}/{actorId}/{path} actors idTellActor
//
// Operation summary.
//
// Operation detailed description
//
//     Consumes: application/json
//     Produces: application/json
//     Schemes: http, https
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

// swagger:route POST /broadcast/{path} utility idBroadcast
//
// Operation summary.
//
// Operation detailed description
//
//     Consumes: application/json
//     Produces: application/json
//     Schemes: http, https
//
func broadcast(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	runtime.Broadcast(ctx, ps.ByName("path"), runtime.ReadAll(r.Body), r.Header.Get("Content-Type"))
	fmt.Fprint(w, "OK")
}

// swagger:route POST /call/{service}/{path} services idCallService
//
// Operation summary.
//
// Operation detailed description
//
//     Consumes: application/json
//     Produces: application/json
//     Schemes: http, https
//

// swagger:route POST /actor-call/{actorType}/{actorId}/{path} actors idCallActor
//
// Operation summary.
//
// Operation detailed description
//
//     Consumes: application/json
//     Produces: application/json
//     Schemes: http, https
//
func call(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var reply *runtime.Reply
	var err error
	if ps.ByName("service") != "" {
		reply, err = runtime.CallService(ctx, ps.ByName("service"), ps.ByName("path"), runtime.ReadAll(r.Body), r.Header.Get("Content-Type"), r.Header.Get("Accept"))
	} else {
		reply, err = runtime.CallActor(ctx, runtime.Actor{Type: ps.ByName("type"), ID: ps.ByName("id")}, ps.ByName("path"), runtime.ReadAll(r.Body), r.Header.Get("Content-Type"), r.Header.Get("Accept"))
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

// swagger:route GET /actor-migrate/{actorType}/{actorId} actors idActorMigrate
//
// Operation summary.
//
// Operation detailed description
//
//     Schemes: http, https
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

// swagger:route POST /actor-reminder/{actorType}/{actorId}/cancel actors idCancelReminder
//
// Cancel all matching reminders.
//
// This operatation cancels reminders for the actor specified in the path.
// If a reminder id is provided as a parameter, only the reminder that
// matches that id will be cancelled. If no id is provided, all
// of the specified actor's reminders will be cancelled.
//
//     Consumes: application/json
//     Produces: application/json
//     Schemes: http, https
//

// swagger:route POST /actor-reminder/{actorType}/{actorId}/get actors idCancelReminder
//
// Get all matching reminders.
//
// This operatation returns all reminders for the actor(s) specified in the path.
// If a reminder id is provided as a parameter, only reminders that
// have that id will be returned.
//
//     Consumes: application/json
//     Produces: application/json
//     Schemes: http, https
//

// swagger:route POST /actor-reminder/{actorType}/{actorId}/schedule actors idScheduleReminder
//
// Schedule a reminder.
//
// This operatation schedules a reminder for the actor specified in the path.
//
//     Consumes: application/json
//     Produces: application/json
//     Schemes: http, https
//
func reminder(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	action := ps.ByName("action")
	if !(action == "cancel" || action == "get" || action == "schedule") {
		http.Error(w, fmt.Sprintf("Invalid action: %v", action), http.StatusBadRequest)
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

// swagger:route POST /actor-state/{actorType}/{actorId}/{key} actors idActorStateSet
//
// Operation summary.
//
// Operation detailed description
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

// swagger:route GET /actor-state-404/{actorType}/{actorId}/{key} actors idActorStateGet404
//
// Operation summary.
//
// Operation detailed description
//
//     Consumes: application/json
//     Produces: application/json
//     Schemes: http, https
//
func get404(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if reply, err := store.HGet(stateKey(ps.ByName("type"), ps.ByName("id")), ps.ByName("key")); err == store.ErrNil {
		http.Error(w, "Not Found", http.StatusNotFound)
	} else if err != nil {
		http.Error(w, fmt.Sprintf("HGET failed: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, reply)
	}
}

// swagger:route GET /actor-state/{actorType}/{actorId}/{key} actors idActorStateGet
//
// Operation summary.
//
// Operation detailed description
//
//     Consumes: application/json
//     Produces: application/json
//     Schemes: http, https
//
func get(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if reply, err := store.HGet(stateKey(ps.ByName("type"), ps.ByName("id")), ps.ByName("key")); err != nil && err != store.ErrNil {
		http.Error(w, fmt.Sprintf("HGET failed: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, reply)
	}
}

// swagger:route DELETE /actor-state/{actorType}/{actorId}/{key} actors idActorStateDeleteKey
//
// Operation summary.
//
// Operation detailed description
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

// swagger:route GET /actor-state/{actorType}/{actorId} actors idActorStateGetAll
//
// Operation summary.
//
// Operation detailed description
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

// swagger:route DELETE /actor-state/{actorType}/{actorId} actors idActorStateDeleteAll
//
// Operation summary.
//
// Operation detailed description
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

// swagger:route GET /kill utility idKill
//
// Operation summary.
//
// Operation detailed description
//
//     Schemes: http, https
//
func kill(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	logger.Info("Invoking cancel() in response to kill request")
	cancel()
	wg.Wait()
	fmt.Fprint(w, "OK")
}

// swagger:route GET /killall utility idKillAll
//
// Operation summary.
//
// Operation detailed description
//
//     Schemes: http, https
//
func killall(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	runtime.KillAll(ctx)
	fmt.Fprint(w, "OK")
	cancel()
}

// swagger:route GET /health utility health
//
// Operation summary.
//
// Operation detailed description
//
//     Schemes: http, https
//
func health(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprint(w, "OK")
}

// server implements the HTTP server
func server(listener net.Listener) {
	router := httprouter.New()
	router.POST("/kar/tell/:service/*path", tell)
	router.POST("/kar/call/:service/*path", call)
	router.POST("/kar/actor-tell/:type/:id/*path", tell)
	router.POST("/kar/actor-call/:type/:id/*path", call)
	router.GET("/kar/actor-migrate/:type/:id", migrate)
	router.POST("/kar/actor-reminder/:type/:id/:action", reminder)
	router.POST("/kar/actor-state/:type/:id/:key", set)
	router.GET("/kar/actor-state-404/:type/:id/:key", get404)
	router.GET("/kar/actor-state/:type/:id/:key", get)
	router.DELETE("/kar/actor-state/:type/:id/:key", del)
	router.GET("/kar/actor-state/:type/:id", getAll)
	router.DELETE("/kar/actor-state/:type/:id", delAll)
	router.GET("/kar/kill", kill)
	router.GET("/kar/killall", killall)
	router.GET("/kar/health", health)
	router.POST("/kar/broadcast/*path", broadcast)
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
