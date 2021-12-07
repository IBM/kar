//
// Copyright IBM Corporation 2020,2021
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

	redis_port, is_present := os.LookupEnv("REDIS_PORT")
	if !is_present {
		log.Print("REDIS_PORT var not set")
	}

	redis_port_integer, err := strconv.Atoi(redis_port)
	if err != nil {
		log.Printf("failed to convert Redis port to an integer value: %v", err)
		os.Exit(1)
	}

	redis_host, is_present := os.LookupEnv("REDIS_HOST")
	if !is_present {
		log.Print("REDIS_HOST var not set")
	}

	sc := &store.StoreConfig{
		MangleKey:         func(s string) string { return s },
		UnmangleKey:       func(s string) string { return s },
		RequestRetryLimit: -1 * time.Second,
		LongOperation:     60 * time.Second,
		Host:              redis_host,
		Port:              redis_port_integer,
	}

	if err := store.Dial(sc); err != nil {
		log.Printf("failed to connect to Redis: %v", err)
		os.Exit(1)
	}

	kafka_version, is_present := os.LookupEnv("KAFKA_VERSION")
	if !is_present {
		log.Print("KAFKA_VERSION var not set")
	}

	kafka_brokers, is_present := os.LookupEnv("KAFKA_BROKERS")
	if !is_present {
		log.Print("KAFKA_BROKERS var not set")
	}

	conf := &rpc.Config{
		Version: kafka_version,
		Brokers: strings.Split(kafka_brokers, ","),
	}

	// start service providing the name of the service
	clientClosed, err := rpc.Connect(c.ClientCtx, appName, conf, "client")
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
