//
// Copyright IBM Corporation 2020,2023
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

package checker

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/kar/core/pkg/logger"
	"github.com/IBM/kar/core/pkg/rpc"
	"github.com/IBM/kar/core/pkg/store"
)

// Connection type
type Connection struct {
	ClientCtx    context.Context
	ClientCancel context.CancelFunc
	ClientClosed <-chan struct{}
	appName      string
}

// ConnectClient --
func (c *Connection) ConnectClient(appName string) {
	c.appName = appName
	logger.SetVerbosity("INFO")

	c.ClientCtx, c.ClientCancel = context.WithCancel(context.Background())

	redisPort, isPresent := os.LookupEnv("REDIS_PORT")
	if !isPresent {
		log.Print("REDIS_PORT var not set")
	}

	redisPortInteger, err := strconv.Atoi(redisPort)
	if err != nil {
		log.Printf("failed to convert Redis port to an integer value: %v", err)
		os.Exit(1)
	}

	redisHost, isPresent := os.LookupEnv("REDIS_HOST")
	if !isPresent {
		log.Print("REDIS_HOST var not set")
	}

	sc := &store.StoreConfig{
		MangleKey:         func(s string) string { return s },
		UnmangleKey:       func(s string) string { return s },
		RequestRetryLimit: -1 * time.Second,
		LongOperation:     60 * time.Second,
		Host:              redisHost,
		Port:              redisPortInteger,
	}

	if err := store.Dial(c.ClientCtx, sc); err != nil {
		log.Printf("failed to connect to Redis: %v", err)
		os.Exit(1)
	}

	kafkaVersion, isPresent := os.LookupEnv("KAFKA_VERSION")
	if !isPresent {
		log.Print("KAFKA_VERSION var not set")
	}

	kafkaBrokers, isPresent := os.LookupEnv("KAFKA_BROKERS")
	if !isPresent {
		log.Print("KAFKA_BROKERS var not set")
	}

	conf := &rpc.Config{
		Version: kafkaVersion,
		Brokers: strings.Split(kafkaBrokers, ","),
	}

	// start service providing the name of the service
	clientClosed, err := rpc.Connect(c.ClientCtx, appName, 0, conf, "client")
	if err != nil {
		log.Printf("failed to connect to Kafka: %v", err)
		os.Exit(1)
	}
	c.ClientClosed = clientClosed
}

// CloseClient --
func (c *Connection) CloseClient() {
	log.Print("success")
	defer store.Close()
	c.ClientCancel()
	<-c.ClientClosed
}
