package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.ibm.com/solsa/kar.git/internal/config"
	"github.ibm.com/solsa/kar.git/internal/pubsub"
	"github.ibm.com/solsa/kar.git/internal/store"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

var (
	serviceURL = fmt.Sprintf("http://127.0.0.1:%d", config.ServicePort)

	// pending requests: map uuids to channel (string -> channel string)
	requests = sync.Map{}

	// termination
	quit = make(chan struct{})
	wg   = sync.WaitGroup{}
)

func text(r io.Reader) string {
	buf, _ := ioutil.ReadAll(r)
	return string(buf)
}

func post(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	service := ps.ByName("service")

	err := pubsub.Send(service, map[string]string{
		"kind":         "post",
		"path":         ps.ByName("path"),
		"content-type": r.Header.Get("Content-Type"),
		"payload":      text(r.Body)})
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to send message to service %s: %v", service, err), http.StatusInternalServerError)
	} else {
		fmt.Fprintln(w, "OK")
	}
}

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
		"origin":       config.ServiceName,
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

func respond(m map[string]string, payload string) {
	err := pubsub.Send(m["origin"], map[string]string{
		"kind":    "reply",
		"id":      m["id"],
		"payload": payload})
	if err != nil {
		logger.Error("failed to reply to request %s from service %s: %v", m["id"], m["origin"], err)
	}
}

func subscriber(channel <-chan map[string]string) {
	defer wg.Done()

	for {
		select {
		case _, _ = <-quit:
			return

		case m := <-channel:
			switch m["kind"] {
			case "post":
				_, err := http.Post(serviceURL+m["path"], m["content-type"], strings.NewReader(m["payload"])) // TODO Accept header
				if err != nil {
					logger.Error("failed to post to %s%s: %v", serviceURL, m["path"], err)
				}

			case "call":
				res, err := http.Post(serviceURL+m["path"], m["content-type"], strings.NewReader(m["payload"]))
				if err != nil {
					logger.Error("failed to post to %s%s: %v", serviceURL, m["path"], err)
					respond(m, "")
				} else {
					respond(m, text(res.Body))
				}

			case "reply":
				if ch, ok := requests.Load(m["id"]); ok {
					ch.(chan string) <- m["payload"]
				}

			default:
				logger.Error("failed to process message with kind %s", m["kind"])
			}
		}
	}
}

func setKey(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if err := store.Set(ps.ByName("key"), text(r.Body)); err != nil {
		http.Error(w, fmt.Sprintf("failed to set key %s: %v", ps.ByName("key"), err), http.StatusInternalServerError)
	} else {
		fmt.Fprintln(w, "OK")
	}
}

func getKey(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	reply, err := store.Get(ps.ByName("key"))
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get key %s: %v", ps.ByName("key"), err), http.StatusInternalServerError)
	} else if reply != nil {
		fmt.Fprintf(w, "%s", *reply)
	} else {
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}

func delKey(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if err := store.Del(ps.ByName("key")); err != nil {
		http.Error(w, fmt.Sprintf("failed to delete key %s: %v", ps.ByName("key"), err), http.StatusInternalServerError)
	} else {
		fmt.Fprintln(w, "OK")
	}
}

func server(listener net.Listener) {
	defer wg.Done()

	router := httprouter.New()

	router.POST("/kar/post/:service/*path", post)
	router.POST("/kar/call/:service/*path", call)

	router.POST("/kar/set/:key", setKey)
	router.GET("/kar/get/:key", getKey)
	router.GET("/kar/del/:key", delKey)

	srv := &http.Server{Handler: router}

	go func() {
		if err := srv.Serve(listener); err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed: %v", err)
		}
	}()

	_, _ = <-quit
	if err := srv.Shutdown(context.Background()); err != nil {
		logger.Fatal("failed to shutdown HTTP server: %v", err)
	}
}

func dump(prefix string, in io.Reader) {
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		log.Printf(prefix+"%s", scanner.Text())
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
		logger.Info("launching service...")

		cmd := exec.Command(args[0], args[1:]...)
		cmd.Env = append(os.Environ(), port1, port2)
		cmd.Stdin = os.Stdin
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			logger.Error("failed to capture stdout from service: %v", err)
		}
		go dump("[STDOUT] ", stdout)
		stderr, err := cmd.StderrPipe()
		if err != nil {
			logger.Error("failed to capture stderr from service: %v", err)
		}
		go dump("[STDERR] ", stderr)

		if err := cmd.Start(); err != nil {
			logger.Error("failed to start service: %v", err)
		}

		if err := cmd.Wait(); err != nil {
			if v, ok := err.(*exec.ExitError); ok {
				logger.Info("service exited with status code %d", v.ExitCode())
			} else {
				logger.Fatal("error waiting for service: %v", err)
			}
		} else {
			logger.Info("service exited normally")
		}

		close(quit)
	}

	wg.Wait()

	logger.Warning("exiting...")
}
