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
	"strconv"
	"strings"
	"sync"

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
	quit = make(chan struct{})
	wg   = sync.WaitGroup{}

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

	err := pubsub.Send(service, map[string]string{
		"kind":         "send",
		"path":         ps.ByName("path"),
		"content-type": r.Header.Get("Content-Type"),
		"payload":      text(r.Body)})
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to send message to service %s: %v", service, err), http.StatusInternalServerError)
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

	id := uuid.New().URN()
	ch := make(chan reply)
	requests.Store(id, ch)
	defer requests.Delete(id)

	err := pubsub.Send(service, map[string]string{
		"kind":         "call",
		"path":         ps.ByName("path"),
		"content-type": r.Header.Get("Content-Type"),
		"accept":       r.Header.Get("Accept"),
		"caller":       config.ServiceName,
		"id":           id,
		"payload":      text(r.Body)})
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to send message to service %s: %v", service, err), http.StatusInternalServerError)
		return
	}

	select {
	case msg := <-ch:
		w.Header().Add("Content-Type", msg.contentType)
		w.WriteHeader(msg.statusCode)
		fmt.Fprint(w, msg.payload)
	case _, _ = <-quit:
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	}
}

// callback sends the result of a call back to the caller
func callback(msg map[string]string, statusCode int, contentType string, payload string) {
	err := pubsub.Send(msg["caller"], map[string]string{
		"kind":         "callback",
		"id":           msg["id"],
		"statusCode":   strconv.Itoa(statusCode),
		"content-type": contentType,
		"payload":      payload})
	if err != nil {
		logger.Error("failed to answer request %s from service %s: %v", msg["id"], msg["caller"], err)
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
	return client.Do(req)
}

// dispatch handles one incoming message
func dispatch(msg map[string]string) {
	defer wg.Done()
	switch msg["kind"] {
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
		if ch, ok := requests.Load(msg["id"]); ok {
			statusCode, _ := strconv.Atoi(msg["statusCode"])
			ch.(chan reply) <- reply{statusCode: statusCode, contentType: msg["content-type"], payload: msg["payload"]}
		} else {
			logger.Error("unexpected callback with id %s", msg["id"])
		}

	default:
		logger.Error("failed to process message with kind %s", msg["kind"])
	}
}

// subscriber dispatches incoming messages to goroutines
func subscriber(channel <-chan map[string]string) {
	defer wg.Done()
	for {
		select {
		case _, _ = <-quit:
			return
		case msg := <-channel:
			wg.Add(1)
			go dispatch(msg)
		}
	}
}

// set route handler
func set(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if err := store.Set(ps.ByName("key"), text(r.Body)); err != nil {
		http.Error(w, fmt.Sprintf("failed to set key: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, "OK")
	}
}

// get route handler
func get(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if reply, err := store.Get(ps.ByName("key")); err != nil {
		http.Error(w, fmt.Sprintf("failed to get key: %v", err), http.StatusInternalServerError)
	} else if reply == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
	} else {
		fmt.Fprint(w, *reply)
	}
}

// del route handler
func del(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if err := store.Del(ps.ByName("key")); err != nil {
		http.Error(w, fmt.Sprintf("failed to delete key: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, "OK")
	}
}

// server implements the HTTP server
func server(listener net.Listener) {
	defer wg.Done()
	router := httprouter.New()
	router.POST("/kar/send/:service/*path", send)
	router.POST("/kar/call/:service/*path", call)
	router.POST("/kar/set/:key", set)
	router.GET("/kar/get/:key", get)
	router.GET("/kar/del/:key", del)
	srv := http.Server{Handler: router}

	go func() {
		if err := srv.Serve(listener); err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed: %v", err)
		}
	}()

	_, _ = <-quit // wait

	if err := srv.Shutdown(context.Background()); err != nil {
		logger.Fatal("failed to shutdown HTTP server: %v", err)
	}
}

func main() {
	logger.Warning("starting...")

	channel := pubsub.Dial()
	defer pubsub.Close()

	store.Dial()
	defer store.Close()

	wg.Add(1)
	go subscriber(channel)

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", config.RuntimePort))
	if err != nil {
		logger.Fatal("Listener failed: %v", err)
	}

	wg.Add(1)
	go server(listener)

	port1 := fmt.Sprintf("KAR_PORT=%d", listener.Addr().(*net.TCPAddr).Port)
	port2 := fmt.Sprintf("KAR_APP_PORT=%d", config.ServicePort)
	logger.Info("%s, %s", port1, port2)

	args := flag.Args()

	if len(args) > 0 {
		launcher.Run(args, append(os.Environ(), port1, port2))
		close(quit)
	}

	wg.Wait()

	logger.Warning("exiting...")
}
