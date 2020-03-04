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

	// pending requests: map uuids to channels (string -> channel string)
	requests = sync.Map{}

	// termination
	quit = make(chan struct{})
	wg   = sync.WaitGroup{}
)

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
		fmt.Fprintln(w, "OK")
	}
}

// call route handler
func call(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	service := ps.ByName("service")

	id := uuid.New().URN()
	ch := make(chan string)
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
	case v := <-ch:
		fmt.Fprint(w, v)
	case _, _ = <-quit:
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	}
}

// callback sends the result of a call back to the caller
func callback(m map[string]string, payload string) {
	err := pubsub.Send(m["caller"], map[string]string{
		"kind":    "callback",
		"id":      m["id"],
		"payload": payload})
	if err != nil {
		logger.Error("failed to answer request %s from service %s: %v", m["id"], m["caller"], err)
	}
}

// subscriber handles incoming messages from pubsub
func subscriber(channel <-chan map[string]string) {
	defer wg.Done()

	for {
		select {
		case _, _ = <-quit:
			return

		case m := <-channel:
			switch m["kind"] {
			case "send":
				_, err := http.Post(serviceURL+m["path"], m["content-type"], strings.NewReader(m["payload"])) // TODO Accept header
				if err != nil {
					logger.Error("failed to post to %s%s: %v", serviceURL, m["path"], err)
				}

			case "call":
				res, err := http.Post(serviceURL+m["path"], m["content-type"], strings.NewReader(m["payload"]))
				if err != nil {
					logger.Error("failed to post to %s%s: %v", serviceURL, m["path"], err)
					callback(m, "") // TODO
				} else {
					callback(m, text(res.Body))
				}

			case "callback":
				if ch, ok := requests.Load(m["id"]); ok {
					ch.(chan string) <- m["payload"]
				}

			default:
				logger.Error("failed to process message with kind %s", m["kind"])
			}
		}
	}
}

// set route handler
func set(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if err := store.Set(ps.ByName("key"), text(r.Body)); err != nil {
		http.Error(w, fmt.Sprintf("failed to set key: %v", err), http.StatusInternalServerError)
	} else {
		fmt.Fprintln(w, "OK")
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
		fmt.Fprintln(w, "OK")
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
