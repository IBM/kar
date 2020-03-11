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

	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
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
	ctx, cancel = context.WithCancel(context.Background())
	wg          = sync.WaitGroup{}

	// http client
	client http.Client
)

func init() {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConnsPerHost = 256
	client = http.Client{Transport: transport}
}

// text converts a request or response body to a string
func text(r io.Reader) string {
	buf, _ := ioutil.ReadAll(r)
	return string(buf)
}

// send route handler
func send(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	service := ps.ByName("service")
	session := ps.ByName("session")

	message := map[string]string{
		"protocol":     "service", // to any sidecar
		"to":           service,   // offering this service
		"command":      "send",    // post with no callback expected
		"path":         ps.ByName("path"),
		"content-type": r.Header.Get("Content-Type"),
		"payload":      text(r.Body)}
	if session != "" {
		message["protocol"] = "session"
		message["session"] = session
	}

	err := pubsub.Send(message)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to send message: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, "OK")
	}
}

type reply struct {
	statusCode  int
	contentType string
	payload     string
}

// call route handler
func call(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	service := ps.ByName("service")
	session := ps.ByName("session")

	request := uuid.New().String()
	ch := make(chan reply)
	requests.Store(request, ch)
	defer requests.Delete(request)

	message := map[string]string{
		"protocol":     "service", // to any sidecar
		"to":           service,   // offering this service
		"command":      "call",    // post expecting a callback with the result
		"path":         ps.ByName("path"),
		"content-type": r.Header.Get("Content-Type"),
		"accept":       r.Header.Get("Accept"),
		"from":         config.ID, // this sidecar
		"request":      request,   // this request
		"payload":      text(r.Body)}
	if session != "" {
		message["protocol"] = "session"
		message["session"] = session
	}

	err := pubsub.Send(message)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to send message: %v", err), http.StatusInternalServerError)
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
}

// callback sends the result of a call back to the caller
func callback(msg map[string]string, statusCode int, contentType string, payload string) {
	err := pubsub.Send(map[string]string{
		"protocol":     "sidecar",   // to a specific
		"to":           msg["from"], // sidecar
		"command":      "callback",
		"request":      msg["request"],
		"statusCode":   strconv.Itoa(statusCode),
		"content-type": contentType,
		"payload":      payload})
	if err != nil {
		logger.Error("failed to answer request %s from service %s: %v", msg["request"], msg["from"], err)
	}
}

// post posts a message to the service
func post(msg map[string]string) (*http.Response, error) {
	req, err := http.NewRequest("POST", serviceURL+msg["path"], strings.NewReader(msg["payload"]))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", msg["content-type"])
	req.Header.Set("Accept", msg["accept"])
	b := backoff.NewExponentialBackOff()
	var res *http.Response
	err = backoff.Retry(func() error {
		res, err = client.Do(req)
		return err
	}, b)
	return res, err
}

// dispatch handles one incoming message
func dispatch(msg map[string]string) {
	defer wg.Done()
	switch msg["command"] {
	case "send":
		res, err := post(msg)
		if err != nil {
			logger.Error("failed to post to %s%s: %v", serviceURL, msg["path"], err)
		} else {
			io.Copy(ioutil.Discard, res.Body)
			res.Body.Close()
		}

	case "call":
		res, err := post(msg)
		if err != nil {
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
			case ch.(chan reply) <- reply{statusCode: statusCode, contentType: msg["content-type"], payload: msg["payload"]}:
			}
		} else {
			logger.Error("unexpected request in callback %s", msg["request"])
		}

	default:
		logger.Error("failed to process message with command %s", msg["command"])
	}
}

// subscriber dispatches incoming messages to goroutines
func subscriber(channel <-chan map[string]string) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-channel:
			wg.Add(1)
			go dispatch(msg)
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

// server implements the HTTP server
func server(listener net.Listener) {
	defer wg.Done()
	router := httprouter.New()
	router.POST("/kar/send/:service/*path", send)
	router.POST("/kar/call/:service/*path", call)
	router.POST("/kar/session/:session/send/:service/*path", send)
	router.POST("/kar/session/:session/call/:service/*path", call)
	router.POST("/kar/set/:key", set)
	router.GET("/kar/get/:key", get)
	router.GET("/kar/del/:key", del)
	srv := http.Server{Handler: router}

	go func() {
		if err := srv.Serve(listener); err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed: %v", err)
		}
	}()

	<-ctx.Done() // wait

	if err := srv.Shutdown(context.Background()); err != nil {
		logger.Fatal("failed to shutdown HTTP server: %v", err)
	}
}

func main() {
	logger.Warning("starting...")

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signals
		cancel()
	}()

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", config.RuntimePort))
	if err != nil {
		logger.Fatal("Listener failed: %v", err)
	}

	channel := pubsub.Dial(ctx)
	defer pubsub.Close()

	store.Dial()
	defer store.Close()

	wg.Add(1)
	go subscriber(channel)

	wg.Add(1)
	go server(listener)

	port1 := fmt.Sprintf("KAR_PORT=%d", listener.Addr().(*net.TCPAddr).Port)
	port2 := fmt.Sprintf("KAR_APP_PORT=%d", config.ServicePort)
	logger.Info("%s %s", port1, port2)

	args := flag.Args()

	if len(args) > 0 {
		launcher.Run(ctx, args, append(os.Environ(), port1, port2))
		cancel()
	}

	wg.Wait()

	logger.Warning("exiting...")
}
