package main

import (
	"context"
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
	var e *actors.Entry
	if msg["protocol"] == "actor" {
		actor := actors.Actor{Type: msg["type"], ID: msg["id"]}
		var fresh bool
		e, fresh = actors.Acquire(ctx, actor)
		if e == nil {
			return ctx.Err()
		}
		defer e.Release()
		if fresh {
			activate(actor)
		}
		msg["path"] = "/actor/" + actor.Type + "/" + actor.ID + msg["path"]
	}

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
	for msg := range channel {
		if !msg.Confirm() {
			continue // message has been or is handled elsewhere
		}
		if msg.Valid { // message is intended for this sidecar
			if dispatch(msg.Value) == nil {
				msg.Mark() // message handled successfully
			}
		} else { // message is intended for another sidecar
			if forward(msg.Value) == nil {
				msg.Mark() // message forwarded successfully
			}
		}
	}
}

// set route handler
func set(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if reply, err := store.Set("state"+config.Separator+ps.ByName("key"), text(r.Body)); err != nil {
		http.Error(w, fmt.Sprintf("failed to set key: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, reply)
	}
}

// get route handler
func get(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if reply, err := store.Get("state" + config.Separator + ps.ByName("key")); err == store.ErrNil {
		http.Error(w, "Not Found", http.StatusNotFound)
	} else if err != nil {
		http.Error(w, fmt.Sprintf("failed to get key: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, reply)
	}
}

// del route handler
func del(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if reply, err := store.Del("state" + config.Separator + ps.ByName("key")); err != nil {
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

func healthTest(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprint(w, "OK")
}

// test scaffolding for reminders.  to be deleted soon.
func reminderTest(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	now := time.Now()
	actors.SchedulePeriodicReminder("hello-10", now.Add(5*time.Second), 10*time.Second)
	actors.SchedulePeriodicReminder("hello-2", now.Add(10*time.Second), 2*time.Second)
}

// server implements the HTTP server
func server(listener net.Listener) {
	router := httprouter.New()
	router.POST("/kar/send/:service/*path", send)
	router.POST("/kar/call/:service/*path", call)
	router.POST("/kar/actor-send/:type/:id/*path", send)
	router.POST("/kar/actor-call/:type/:id/*path", call)
	router.POST("/kar/set/:key", set)
	router.GET("/kar/get/:key", get)
	router.GET("/kar/del/:key", del)
	router.GET("/kar/kill", kill)
	router.GET("/kar/killall", killall)
	router.GET("/kar/health", healthTest)
	router.POST("/kar/broadcast/*path", broadcast)
	// TEMP: dummy route for reminders
	router.GET("/kar/reminder/testme", reminderTest)
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
