package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.ibm.com/solsa/kar.git/internal/actors"
	"github.ibm.com/solsa/kar.git/internal/commands"
	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/internal/launcher"
	"github.ibm.com/solsa/kar.git/internal/proxy"
	"github.ibm.com/solsa/kar.git/internal/pubsub"
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

// send route handler
func send(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var err error
	if ps.ByName("service") != "" {
		err = commands.TellService(ctx, ps.ByName("service"), ps.ByName("path"), proxy.Read(r.Body), r.Header.Get("Content-Type"))
	} else {
		err = commands.TellActor(ctx, actors.Actor{Type: ps.ByName("type"), ID: ps.ByName("id")}, ps.ByName("path"), proxy.Read(r.Body), r.Header.Get("Content-Type"))
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

// broadcast route handler
func broadcast(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	commands.Broadcast(ctx, ps.ByName("path"), proxy.Read(r.Body), r.Header.Get("Content-Type"))
	fmt.Fprint(w, "OK")
}

// call route handler
func call(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var reply *commands.Reply
	var err error
	if ps.ByName("service") != "" {
		reply, err = commands.CallService(ctx, ps.ByName("service"), ps.ByName("path"), proxy.Read(r.Body), r.Header.Get("Content-Type"), r.Header.Get("Accept"))
	} else {
		reply, err = commands.CallActor(ctx, actors.Actor{Type: ps.ByName("type"), ID: ps.ByName("id")}, ps.ByName("path"), proxy.Read(r.Body), r.Header.Get("Content-Type"), r.Header.Get("Accept"))
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

// migrate route handler
func migrate(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	commands.Migrate(ctx, actors.Actor{Type: ps.ByName("type"), ID: ps.ByName("id")})
	fmt.Fprint(w, "OK")
}

// subscriber handles incoming messages
func subscriber(channel <-chan pubsub.Message) {
	for m := range channel {
		commands.Process(ctx, cancel, m)
	}
}

func mangle(key string) string {
	return "main" + config.Separator + "state" + config.Separator + key
}

// reminder route handler
// Supported paths: "/kar/actor-reminder/:type/:id/:action"
//    :type is an actor type
//    :id is an actor id
//    :action is one of: cancel, get, schedule
//
// The body of the request is a JSON object with the following format
//    cancel: { id:string }
//    get: { id:string }
//    schedule: { id:string, path:string, deadline:string(ISO-8601) period:string(ISO-8601), data: any}
func reminder(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	actorType := ps.ByName("type")
	actorID := ps.ByName("id")
	action := ps.ByName("action")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch action {
	case "cancel":
		var payload actors.CancelReminderPayload
		err = json.Unmarshal(body, &payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		found, err := actors.CancelReminder(actorType, actorID, payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		if found {
			fmt.Fprintf(w, "OK")
		} else {
			http.Error(w, "Not Found", http.StatusNotFound)
		}

	case "get":
		var payload actors.GetReminderPayload
		err = json.Unmarshal(body, &payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		reminders, err := actors.GetReminders(actorType, actorID, payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(reminders)

	case "schedule":
		var payload actors.ScheduleReminderPayload
		err = json.Unmarshal(body, &payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		validRequest, err := actors.ScheduleReminder(actorType, actorID, payload)
		if err != nil {
			if validRequest {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			} else {
				http.Error(w, err.Error(), http.StatusBadRequest)
			}
		}
		fmt.Fprintf(w, "OK")

	default:
		http.Error(w, fmt.Sprintf("Invalid action: %v", action), http.StatusBadRequest)
	}

}

// set route handler
func set(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if reply, err := store.Set(mangle(ps.ByName("key")), proxy.Read(r.Body)); err != nil {
		http.Error(w, fmt.Sprintf("failed to set key: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, reply)
	}
}

// get route handler
func get(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if reply, err := store.Get(mangle(ps.ByName("key"))); err == store.ErrNil {
		http.Error(w, "Not Found", http.StatusNotFound)
	} else if err != nil {
		http.Error(w, fmt.Sprintf("failed to get key: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, reply)
	}
}

// del route handler
func del(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if reply, err := store.Del(mangle(ps.ByName("key"))); err != nil {
		http.Error(w, fmt.Sprintf("failed to delete key: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, reply)
	}
}

// kill route handler
func kill(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	logger.Info("Invoking cancel() in response to kill request")
	cancel()
	wg.Wait()
	fmt.Fprint(w, "OK")
}

func killall(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	commands.KillAll(ctx)
	fmt.Fprint(w, "OK")
	cancel()
}

// implement sidecar's livenessProbe for Kubernetes
func health(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprint(w, "OK")
}

// server implements the HTTP server
func server(listener net.Listener) {
	router := httprouter.New()
	router.POST("/kar/send/:service/*path", send)
	router.POST("/kar/call/:service/*path", call)
	router.POST("/kar/actor-send/:type/:id/*path", send)
	router.POST("/kar/actor-call/:type/:id/*path", call)
	router.GET("/kar/actor-migrate/:type/:id", migrate)
	router.POST("/kar/actor-reminder/:type/:id/:action", reminder)
	router.POST("/kar/set/:key", set)
	router.GET("/kar/get/:key", get)
	router.GET("/kar/del/:key", del)
	router.GET("/kar/kill", kill)
	router.GET("/kar/killall", killall)
	router.GET("/kar/health", health)
	router.POST("/kar/broadcast/*path", broadcast)
	srv := http.Server{Handler: router}

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
		logger.Fatal("Listener failed: %v", err)
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
		ticker := time.NewTicker(10 * time.Second)
		for {
			select {
			case now := <-ticker.C:
				logger.Debug("starting collection")
				actors.Collect(ctx, now.Add(-10*time.Second), commands.Deactivate) // TODO invoke deactivate route
				logger.Debug("finishing collection")
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(config.ActorReminderInterval)
		for {
			select {
			case now := <-ticker.C:
				actors.ProcessReminders(ctx, now)
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()

	port1 := fmt.Sprintf("KAR_PORT=%d", listener.Addr().(*net.TCPAddr).Port)
	port2 := fmt.Sprintf("KAR_APP_PORT=%d", config.ServicePort)
	logger.Info("%s %s", port1, port2)

	args := flag.Args()

	if len(args) > 0 {
		exitCode = launcher.Run(ctx9, args, append(os.Environ(), port1, port2))
		cancel()
	}

	wg.Wait()

	<-finished

	logger.Warning("exiting...")
}
