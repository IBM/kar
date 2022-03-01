//
// Copyright IBM Corporation 2020,2022
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

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
	"strings"
	"sync"
	"syscall"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/IBM/kar/core/internal/config"
	"github.com/IBM/kar/core/pkg/logger"
	"github.com/IBM/kar/core/pkg/rpc"
	"github.com/IBM/kar/core/pkg/store"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	// prometheus metrics endpoint
	router.GET("/metrics", handler2Handle(promhttp.Handler()))

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
	router.POST(base+"/actor/:type/:id/state/:key", routeImplSubmapOps)
	router.GET(base+"/actor/:type/:id/state", routeImplGetAll)
	router.POST(base+"/actor/:type/:id/state", routeImplStateUpdate)
	router.DELETE(base+"/actor/:type/:id/state", routeImplDelAll)
	router.DELETE(base+"/actor/:type/:id", routeImplDelActor)

	// kar system methods
	router.GET(base+"/system/health", routeImplHealth)
	router.POST(base+"/system/shutdown", routeImplShutdown)
	router.GET(base+"/system/information/:component", routeImplGetInformation)

	// events
	router.POST(base+"/event/:topic/publish", routeImplPublish)
	router.DELETE(base+"/event/:topic", routeImplDeleteTopic)
	router.PUT(base+"/event/:topic", routeImplCreateTopic)

	return http.Server{Handler: h2c.NewHandler(router, &http2.Server{MaxConcurrentStreams: 262144})}
}

func handler2Handle(h http.Handler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		h.ServeHTTP(w, r)
	}
}

// Main is the main entrypoint for the KAR runtime
func Main() {
	logger.Warning("starting...")
	logger.Info("redis: %v:%v", config.RedisConfig.Host, config.RedisConfig.Port)
	logger.Info("kafka: %v", strings.Join(config.KafkaConfig.Brokers, ","))
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
		logger.Fatal("TCP listener failed: %v", err)
	}

	redisConfig := config.RedisConfig
	redisConfig.MangleKey = func(key string) string { return "kar" + config.Separator + config.AppName + config.Separator + key }
	redisConfig.UnmangleKey = func(key string) string {
		parts := strings.Split(key, config.Separator)
		if parts[0] == "kar" && parts[1] == config.AppName {
			return strings.Join(parts[2:], config.Separator)
		}
		return key
	}
	redisConfig.RequestRetryLimit = config.RequestRetryLimit

	if err = store.Dial(&redisConfig); err != nil {
		logger.Fatal("failed to connect to Redis: %v", err)
	}
	defer store.Close()

	// Connecting to Kafka takes a long time and can disrupt the application when we re-partition.
	// Recognize the command combinations that do not require Kafka and short-circuit.
	requiresPubSub := true
	if config.CmdName == config.GetCmd && config.GetSystemComponent == "actors" && !config.GetResidentOnly {
		requiresPubSub = false
	}

	topic := "kar" + config.Separator + config.AppName

	if config.CmdName == config.PurgeCmd {
		purge(topic, "*")
		return
	} else if config.CmdName == config.DrainCmd {
		purge(topic, "pubsub"+config.Separator+"*")
		return
	}

	// Connect to Kafka
	var closed <-chan struct{} = nil
	if requiresPubSub {
		myServices := append([]string{config.ServiceName}, config.ActorTypes...)
		closed, err = rpc.Connect(ctx, topic, &config.KafkaConfig, myServices...)
		if err != nil {
			logger.Fatal("failed to connect to Kafka: %v", err)
		}
		karPublisher, err = rpc.NewPublisher(&config.KafkaConfig)
		if err != nil {
			logger.Fatal("failed to create event publisher: %v", err)
		}
		defer karPublisher.Close()
	}

	args := flag.Args()

	if config.CmdName == config.InvokeCmd {
		exitCode = invokeActorMethod(ctx9, args)
		cancel()
	} else if config.CmdName == config.RestCmd {
		exitCode = invokeServiceEndpoint(ctx9, args)
		cancel()
	} else if config.CmdName == config.GetCmd {
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
		requestTimeout := fmt.Sprintf("KAR_REQUEST_TIMEOUT=%d", config.RequestRetryLimit.Milliseconds())
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

		wg.Add(1)
		go func() {
			defer wg.Done()
			ValidateActorConfig(ctx)
		}()

		if len(args) > 0 {
			exitCode = Run(ctx9, args, append(os.Environ(), runtimePort, appPort, requestTimeout))
			cancel()
		}
	}

	if requiresPubSub {
		<-closed // first wait for rpc library to shutdown
	}
	wg.Wait() // next wait for the rest of the runtime to shutdown

	cancel9()

	wg9.Wait()

	logger.Warning("exiting...")
}

func purge(topic string, pattern string) {
	if err := rpc.DeleteTopic(&config.KafkaConfig, topic); err != nil {
		logger.Error("failed to delete Kafka topic: %v", err)
	}
	if count, err := store.Purge(ctx, pattern); err != nil {
		logger.Error("failed to delete Redis keys: %v", err)
	} else {
		logger.Info("%v deleted keys", count)
	}
}
