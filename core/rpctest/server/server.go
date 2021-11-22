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

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/IBM/kar/core/pkg/logger"
	"github.com/IBM/kar/core/pkg/rpc"
	"github.com/IBM/kar/core/pkg/store"
	// rpclib "github.com/IBM/kar/core/pkg/rpc"
)

var ctx, cancel = context.WithCancel(context.Background())

func incr(ctx context.Context, t rpc.Target, v []byte) (*rpc.Destination, []byte, error) {
	fmt.Println("BLA")
	return nil, []byte{v[0] + 1}, nil
}

func fail(ctx context.Context, t rpc.Target, v []byte) (*rpc.Destination, []byte, error) {
	return nil, nil, errors.New("failed")
}

func exit(ctx context.Context, t rpc.Target, v []byte) (*rpc.Destination, []byte, error) {
	log.Printf("%s", v)
	cancel()
	return nil, nil, nil
}

func main() {
	logger.SetVerbosity("INFO")

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
	// defer store.Close()

	conf := &rpc.Config{
		Version: "2.8.0",
		Brokers: []string{"localhost:31093"},
	}

	// rpclib.PlacementCache = false

	// register function on this service
	rpc.Register("incr", incr)
	rpc.Register("fail", fail)
	rpc.Register("exit", exit)

	// start service
	closed, err := rpc.Connect(ctx, "test-rpc", conf, "server", "actor")
	if err != nil {
		log.Printf("failed to connect to Kafka: %v", err)
		os.Exit(1)
	}

	<-closed
}
