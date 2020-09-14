//go:generate swagger generate spec

package main

/*
 * This file contains the top-level control flow for the kar cli
 */

import (
	"context"
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
	"github.ibm.com/solsa/kar.git/core/internal/config"
	"github.ibm.com/solsa/kar.git/core/internal/pubsub"
	"github.ibm.com/solsa/kar.git/core/internal/runtime"
	"github.ibm.com/solsa/kar.git/core/internal/store"
	"github.ibm.com/solsa/kar.git/core/pkg/logger"
)

var (
	// termination
	ctx9, cancel9 = context.WithCancel(context.Background()) // preemptive: kill subprocess
	ctx, cancel   = context.WithCancel(ctx9)                 // cooperative: wait for subprocess
	wg            = &sync.WaitGroup{}                        // wait for kafka consumer and http server to stop processing requests
	wg9           = &sync.WaitGroup{}                        // wait for signal handler
)

// server implements the HTTP server
func server(listener net.Listener) http.Server {
	base := "/kar/v1"
	router := httprouter.New()
	methods := [7]string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}

	// service invocation - handles all common HTTP requests
	for _, method := range methods {
		router.Handle(method, base+"/service/:service/call/*path", call)
	}

	// callbacks
	router.POST(base+"/await", awaitPromise)

	// actor invocation
	router.POST(base+"/actor/:type/:id/call/*path", call)

	// reminders
	router.GET(base+"/actor/:type/:id/reminders/:reminderId", reminder)
	router.GET(base+"/actor/:type/:id/reminders", reminder)
	router.PUT(base+"/actor/:type/:id/reminders/:reminderId", reminder)
	router.DELETE(base+"/actor/:type/:id/reminders/:reminderId", reminder)
	router.DELETE(base+"/actor/:type/:id/reminders", reminder)

	// events
	router.GET(base+"/actor/:type/:id/events/:subscriptionId", subscription)
	router.GET(base+"/actor/:type/:id/events", subscription)
	router.PUT(base+"/actor/:type/:id/events/:subscriptionId", subscription)
	router.DELETE(base+"/actor/:type/:id/events/:subscriptionId", subscription)
	router.DELETE(base+"/actor/:type/:id/events", subscription)

	// actor state
	router.GET(base+"/actor/:type/:id/state/:key/:subkey", get)
	router.PUT(base+"/actor/:type/:id/state/:key/:subkey", set)
	router.DELETE(base+"/actor/:type/:id/state/:key/:subkey", del)
	router.HEAD(base+"/actor/:type/:id/state/:key/:subkey", containsKey)
	router.GET(base+"/actor/:type/:id/state/:key", get)
	router.PUT(base+"/actor/:type/:id/state/:key", set)
	router.DELETE(base+"/actor/:type/:id/state/:key", del)
	router.HEAD(base+"/actor/:type/:id/state/:key", containsKey)
	router.POST(base+"/actor/:type/:id/state/:key", mapOps)
	router.GET(base+"/actor/:type/:id/state", getAll)
	router.POST(base+"/actor/:type/:id/state", setMultiple)
	router.DELETE(base+"/actor/:type/:id/state", delAll)

	// kar system methods
	router.GET(base+"/system/health", health)
	router.POST(base+"/system/shutdown", shutdown)
	router.POST(base+"/system/post", post)
	router.GET(base+"/system/information/:component", getInformation)

	// events
	router.POST(base+"/event/:topic/publish", publish)
	router.PUT(base+"/event/:topic/", createTopic)
	router.DELETE(base+"/event/:topic/", deleteTopic)

	return http.Server{Handler: h2c.NewHandler(router, &http2.Server{MaxConcurrentStreams: 262144})}
}

// process incoming message asynchronously
// one goroutine, incr and decr WaitGroup
func process(m pubsub.Message) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		runtime.Process(ctx, cancel, m)
	}()
}

func main() {
	logger.Warning("starting...")
	exitCode := 0
	defer func() { os.Exit(exitCode) }()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	wg9.Add(1)
	go func() {
		defer wg9.Done()
		select {
		case <-signals:
			logger.Info("Invoking cancel9() from signal handler")
			cancel9()
		case <-ctx9.Done():
		}
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

	if config.Purge {
		purge("*")
		return
	} else if config.Drain {
		purge("pubsub" + config.Separator + "*")
		return
	}

	// one goroutine, defer close(closed)
	closed, err := pubsub.Join(ctx, process, listener.Addr().(*net.TCPAddr).Port)
	if err != nil {
		logger.Fatal("join failed: %v", err)
	}

	args := flag.Args()

	if config.Invoke {
		exitCode = runtime.Invoke(ctx9, args)
		cancel()
	} else if config.Get != "" {
		exitCode = runtime.GetInformation(ctx9, args)
		cancel()
	} else {
		// start server and background tasks
		srv := server(listener)

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
			runtime.CloseIdleConnections()
		}()

		runtimePort := fmt.Sprintf("KAR_RUNTIME_PORT=%d", listener.Addr().(*net.TCPAddr).Port)
		appPort := fmt.Sprintf("KAR_APP_PORT=%d", config.AppPort)
		requestTimeout := fmt.Sprintf("KAR_REQUEST_TIMEOUT=%d", config.RequestTimeout.Milliseconds())
		logger.Info("%s %s", runtimePort, appPort)

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
			runtime.ManageBindings(ctx)
		}()

		if len(args) > 0 {
			exitCode = runtime.Run(ctx9, args, append(os.Environ(), runtimePort, appPort, requestTimeout))
			cancel()
		}
	}

	<-closed // wait for closed consumer first since process adds to WaitGroup
	wg.Wait()

	cancel9()

	wg9.Wait()

	logger.Warning("exiting...")
}

func purge(pattern string) {
	if err := pubsub.Purge(); err != nil {
		logger.Error("failed to delete Kafka topic: %v", err)
	}
	if count, err := store.Purge(pattern); err != nil {
		logger.Error("failed to delete Redis keys: %v", err)
	} else {
		logger.Info("%v deleted keys", count)
	}
}
