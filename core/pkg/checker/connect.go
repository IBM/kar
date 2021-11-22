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
	"time"

	"github.com/IBM/kar/core/pkg/logger"
	"github.com/IBM/kar/core/pkg/store"
	"github.com/IBM/kar/core/pkg/rpc"
)

// Connection type
type Connection struct {
	ClientCtx context.Context
	ClientCancel context.CancelFunc
	ClientClosed <-chan struct {}
	appName string
}

// ConnectClient --
func (c *Connection) ConnectClient(appName string) {
	c.appName = appName
	logger.SetVerbosity("INFO")

	c.ClientCtx, c.ClientCancel = context.WithCancel(context.Background())

	sc := &store.StoreConfig{
		MangleKey:         func(s string) string { return s },
		UnmangleKey:       func(s string) string { return s },
		RequestRetryLimit: -1 * time.Second,
		LongOperation:     60 * time.Second,
		Host:              "localhost",
		Port:              31379,
	}

	if err := store.Dial(sc); err != nil {
		log.Printf("failed to connect to Reddis: %v", err)
		os.Exit(1)
	}
	defer store.Close()

	conf := &rpc.Config{
		Version: "2.8.0",
		Brokers: []string{"localhost:31093"},
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
	c.ClientCancel()
	<-c.ClientClosed
}