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
	"log"
	"os"
	"time"

	"github.com/IBM/kar/core/pkg/rpc"
	"github.com/IBM/kar/core/pkg/checker"
)

func main() {
	var c checker.Connection
	c.ConnectClient("test-rpc")

	// The remote method to be called on the server.
	destinationIncr := rpc.Destination{Target: rpc.Service{Name: "server"}, Method: "incr"}

	// Send an async request:
	log.Print("async test")
	_, rp, err := rpc.Async(c.ClientCtx, destinationIncr, time.Time{}, []byte{42})
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	// Response from async method:
	response := <-rp
	if response.Err != nil {
		log.Print("async await test failed")
		os.Exit(1)
	} else {
		log.Print("async await successful")
	}

	// Check value is correct:
	log.Print("async result: ", response.Value[0])

	c.CloseClient()
}