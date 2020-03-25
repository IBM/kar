package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.ibm.com/solsa/kar.git/internal/actors"
	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/internal/launcher"
	"github.ibm.com/solsa/kar.git/internal/pubsub"
	"github.ibm.com/solsa/kar.git/internal/store"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

var (
	// service url
	serviceURL = fmt.Sprintf("http://127.0.0.1:%d", config.ServicePort)

	// pending requests: map uuids to channels
	requests = sync.Map{}

	// termination
	ctx9, cancel9 = context.WithCancel(context.Background()) // preemptive: kill subprocess
	ctx, cancel   = context.WithCancel(ctx9)                 // cooperative: wait for subprocess
	wg            = &sync.WaitGroup{}                        // wait for kafka consumer and http server to stop processing requests
	finished      = make(chan struct{})                      // wait for http server to complete shutdown

	// http client
	client http.Client
)

func init() {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConnsPerHost = 256
	client = http.Client{Transport: transport} // TODO adjust timeout
}

// text converts a request or response body to a string
func text(r io.Reader) string {
	buf, _ := ioutil.ReadAll(r)
	return string(buf)
}

// send route handler
func send(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	msg := map[string]string{
		"command":      "send", // post with no callback expected
		"path":         ps.ByName("path"),
		"content-type": r.Header.Get("Content-Type"),
		"payload":      text(r.Body)}
	if ps.ByName("service") != "" {
		msg["protocol"] = "service"
		msg["service"] = ps.ByName("service")
	} else {
		msg["protocol"] = "actor"
		msg["type"] = ps.ByName("type")
		msg["id"] = ps.ByName("id")
	}
	if err := pubsub.Send(msg); err != nil {
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
	payload := text(r.Body)
	for _, sidecar := range pubsub.Sidecars() {
		if sidecar != config.ID { // send to all other sidecars
			pubsub.Send(map[string]string{ // TODO log errors, reuse message object?
				"protocol":     "sidecar",
				"sidecar":      sidecar,
				"command":      "send", // post with no callback expected
				"path":         ps.ByName("path"),
				"content-type": r.Header.Get("Content-Type"),
				"payload":      payload})
		}
	}
	fmt.Fprint(w, "OK")
}

type reply struct {
	statusCode  int
	contentType string
	payload     string
}

// call route handler
func call(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	request := uuid.New().String()
	ch := make(chan reply)
	requests.Store(request, ch)

	msg := map[string]string{
		"command":      "call", // post expecting a callback with the result
		"path":         ps.ByName("path"),
		"content-type": r.Header.Get("Content-Type"),
		"accept":       r.Header.Get("Accept"),
		"from":         config.ID, // this sidecar
		"request":      request,   // this request
		"payload":      text(r.Body)}
	if ps.ByName("service") != "" {
		msg["protocol"] = "service"
		msg["service"] = ps.ByName("service")
	} else {
		msg["protocol"] = "actor"
		msg["type"] = ps.ByName("type")
		msg["id"] = ps.ByName("id")
	}
	if err := pubsub.Send(msg); err != nil {
		if ctx.Err() != nil {
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		} else {
			http.Error(w, fmt.Sprintf("failed to send message: %v", err), http.StatusInternalServerError)
		}
		requests.Delete(request)
		return
	}

	select {
	case msg := <-ch:
		w.Header().Add("Content-Type", msg.contentType)
		w.WriteHeader(msg.statusCode)
		fmt.Fprint(w, msg.payload)
	case <-ctx.Done():
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	}
	requests.Delete(request)
}

// migrate route handler
func migrate(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	actors.Migrate(ctx, actors.Actor{Type: ps.ByName("type"), ID: ps.ByName("id")}, deactivate)
	fmt.Fprint(w, "OK")
}

// callback sends the result of a call back to the caller
func callback(msg map[string]string, statusCode int, contentType string, payload string) {
	err := pubsub.Send(map[string]string{
		"protocol":     "sidecar",
		"sidecar":      msg["from"],
		"command":      "callback",
		"request":      msg["request"],
		"statusCode":   strconv.Itoa(statusCode),
		"content-type": contentType,
		"payload":      payload})
	if err != nil {
		logger.Error("failed to answer request %s from sidecar %s: %v", msg["request"], msg["from"], err)
	}
}

// post posts a message to the service
func httpRequest(method string, msg map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, serviceURL+msg["path"], strings.NewReader(msg["payload"]))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", msg["content-type"])
	req.Header.Set("Accept", msg["accept"])
	var res *http.Response
	err = backoff.Retry(func() error {
		res, err = client.Do(req)
		return err
	}, backoff.WithContext(backoff.NewExponentialBackOff(), ctx)) // TODO adjust timeout
	return res, err
}

func activate(actor actors.Actor) {
	logger.Debug("activating actor %v", actor)
	res, err := httpRequest("GET", map[string]string{"path": "/actor/" + actor.Type + "/" + actor.ID})
	if err != nil {
		logger.Error("failed to activate actor %v: %v", actor, err)
	} else if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNotFound {
		logger.Error("failed to activate actor %v: %s", actor, text(res.Body))
	} else {
		io.Copy(ioutil.Discard, res.Body)
		res.Body.Close()
	}
}

func deactivate(actor actors.Actor) {
	logger.Info("deactivating actor %v", actor)
	res, err := httpRequest("DELETE", map[string]string{"path": "/actor/" + actor.Type + "/" + actor.ID})
	if err != nil {
		logger.Error("failed to deactivate actor %v: %v", actor, err)
	} else if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNotFound {
		logger.Error("failed to deactivate actor %v: %s", actor, text(res.Body))
	} else {
		io.Copy(ioutil.Discard, res.Body)
		res.Body.Close()
	}
}

// dispatch handles one incoming message
func dispatch(msg map[string]string) error {
	switch msg["command"] {
	case "send":
		res, err := httpRequest("POST", msg)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			logger.Error("failed to post to %s%s: %v", serviceURL, msg["path"], err)
		} else {
			io.Copy(ioutil.Discard, res.Body)
			res.Body.Close()
		}

	case "call":
		res, err := httpRequest("POST", msg)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			logger.Error("failed to post to %s%s: %v", serviceURL, msg["path"], err)
			callback(msg, http.StatusBadGateway, "text/plain", "Bad Gateway")
		} else {
			payload := text(res.Body)
			res.Body.Close()
			callback(msg, res.StatusCode, res.Header.Get("Content-Type"), payload)
		}

	case "callback":
		if ch, ok := requests.Load(msg["request"]); ok {
			statusCode, _ := strconv.Atoi(msg["statusCode"])
			select {
			case <-ctx.Done():
				return ctx.Err()
			case ch.(chan reply) <- reply{statusCode: statusCode, contentType: msg["content-type"], payload: msg["payload"]}:
			}
		} else {
			logger.Error("unexpected request in callback %s", msg["request"])
		}

	case "kill":
		cancel()

	default:
		logger.Error("failed to process message with command %s", msg["command"])
	}

	return nil
}

// forward handles misdirected messages due to rebalance
func forward(msg map[string]string) error {
	switch msg["protocol"] {
	case "service": // route to service
		logger.Info("forwarding message to service %s", msg["service"])
	case "actor": // route to actor
		logger.Info("forwarding message to actor %v", actors.Actor{Type: msg["type"], ID: msg["id"]})
	case "sidecar": // route to sidecar
		logger.Info("forwarding message to sidecar %s", msg["sidecar"])
	}
	if err := pubsub.Send(msg); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		switch msg["protocol"] {
		case "service": // route to service
			logger.Error("failed to forward message to service %s: %v", msg["service"], err)
		case "actor": // route to actor
			logger.Error("failed to forward message to actor %v: %v", actors.Actor{Type: msg["type"], ID: msg["id"]}, err)
		case "sidecar": // route to sidecar
			logger.Debug("failed to forward message to sidecar %s: %v", msg["sidecar"], err) // not an error
		}
	}
	return nil
}

// subscriber handles incoming messages
func subscriber(channel <-chan pubsub.Message) {
	for m := range channel {
		invoke(m)
	}
}

func invoke(m pubsub.Message) {
	msg := m.Value
	var err error
	switch msg["protocol"] {
	case "service":
		if msg["service"] == config.ServiceName {
			err = dispatch(msg)
		} else {
			err = forward(msg)
		}
	case "actor":
		err = invokeActor(msg)
	case "sidecar":
		if msg["sidecar"] == config.ID {
			err = dispatch(msg)
		} else {
			err = forward(msg)
		}
	}
	if err == nil {
		m.Mark()
	}
}

func invokeActor(msg map[string]string) error {
	actor := actors.Actor{Type: msg["type"], ID: msg["id"]}
	e, fresh, _ := actors.Acquire(ctx, actor)
	if e == nil && ctx.Err() == nil {
		return forward(msg)
	}
	defer e.Release()
	if fresh {
		activate(actor)
	}
	msg["path"] = "/actor/" + actor.Type + "/" + actor.ID + msg["path"]
	return dispatch(msg)
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
//    schedule: { id:string, entrypoint:string, deadline:string(ISO-8601) period:string(ISO-8601), data: any}
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
	if reply, err := store.Set(mangle(ps.ByName("key")), text(r.Body)); err != nil {
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
	for _, sidecar := range pubsub.Sidecars() {
		if sidecar != config.ID { // send to all other sidecars
			pubsub.Send(map[string]string{ // ignore errors?
				"protocol": "sidecar",
				"sidecar":  sidecar,
				"command":  "kill",
			})
		}
	}
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
				actors.Collect(ctx, now.Add(-10*time.Second), deactivate) // TODO invoke deactivate route
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
		ticker := time.NewTicker(1 * time.Second)
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
