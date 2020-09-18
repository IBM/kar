//go:generate swagger generate spec

package runtime

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
		router.Handle(method, base+"/service/:service/call/*path", routeImplCall)
	}

	// callbacks
	router.POST(base+"/await", routeImplAwaitPromise)

	// actor invocation
	router.POST(base+"/actor/:type/:id/call/*path", routeImplCall)

	// reminders
	router.GET(base+"/actor/:type/:id/reminders/:reminderId", routeImplReminder)
	router.GET(base+"/actor/:type/:id/reminders", routeImplReminder)
	router.PUT(base+"/actor/:type/:id/reminders/:reminderId", routeImplReminder)
	router.DELETE(base+"/actor/:type/:id/reminders/:reminderId", routeImplReminder)
	router.DELETE(base+"/actor/:type/:id/reminders", routeImplReminder)

	// events
	router.GET(base+"/actor/:type/:id/events/:subscriptionId", routeImplSubscription)
	router.GET(base+"/actor/:type/:id/events", routeImplSubscription)
	router.PUT(base+"/actor/:type/:id/events/:subscriptionId", routeImplSubscription)
	router.DELETE(base+"/actor/:type/:id/events/:subscriptionId", routeImplSubscription)
	router.DELETE(base+"/actor/:type/:id/events", routeImplSubscription)

	// actor state
	router.GET(base+"/actor/:type/:id/state/:key/:subkey", routeImplGet)
	router.PUT(base+"/actor/:type/:id/state/:key/:subkey", routeImplSet)
	router.DELETE(base+"/actor/:type/:id/state/:key/:subkey", routeImplDel)
	router.HEAD(base+"/actor/:type/:id/state/:key/:subkey", routeImplContainsKey)
	router.GET(base+"/actor/:type/:id/state/:key", routeImplGet)
	router.PUT(base+"/actor/:type/:id/state/:key", routeImplSet)
	router.DELETE(base+"/actor/:type/:id/state/:key", routeImplDel)
	router.HEAD(base+"/actor/:type/:id/state/:key", routeImplContainsKey)
	router.POST(base+"/actor/:type/:id/state/:key", routeImplMapOps)
	router.GET(base+"/actor/:type/:id/state", routeImplGetAll)
	router.POST(base+"/actor/:type/:id/state", routeImplSetMultiple)
	router.DELETE(base+"/actor/:type/:id/state", routeImplDelAll)

	// kar system methods
	router.GET(base+"/system/health", routeImplHealth)
	router.POST(base+"/system/shutdown", routeImplShutdown)
	router.POST(base+"/system/post", routeImplPost)
	router.GET(base+"/system/information/:component", routeImplGetInformation)

	// events
	router.POST(base+"/event/:topic/publish", routeImplPublish)
	router.PUT(base+"/event/:topic/", routeImplCreateTopic)
	router.DELETE(base+"/event/:topic/", routeImplDeleteTopic)

	return http.Server{Handler: h2c.NewHandler(router, &http2.Server{MaxConcurrentStreams: 262144})}
}

// process incoming message asynchronously
// one goroutine, incr and decr WaitGroup
func process(m pubsub.Message) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		Process(ctx, cancel, m)
	}()
}

// Main is the main entrypoint for the KAR runtime
func Main() {
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
		exitCode = invokeActorMethod(ctx9, args)
		cancel()
	} else if config.Get != "" {
		exitCode = getInformation(ctx9, args)
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
			CloseIdleConnections()
		}()

		runtimePort := fmt.Sprintf("KAR_RUNTIME_PORT=%d", listener.Addr().(*net.TCPAddr).Port)
		appPort := fmt.Sprintf("KAR_APP_PORT=%d", config.AppPort)
		requestTimeout := fmt.Sprintf("KAR_REQUEST_TIMEOUT=%d", config.RequestTimeout.Milliseconds())
		logger.Info("%s %s", runtimePort, appPort)

		wg.Add(1)
		go func() {
			defer wg.Done()
			Collect(ctx)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			ProcessReminders(ctx)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			ManageBindings(ctx)
		}()

		if len(args) > 0 {
			exitCode = Run(ctx9, args, append(os.Environ(), runtimePort, appPort, requestTimeout))
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
