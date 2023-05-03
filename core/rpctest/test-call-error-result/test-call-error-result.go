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

package main

import (
	"log"
	"os"
	"time"

	"github.com/IBM/kar/core/pkg/checker"
	"github.com/IBM/kar/core/pkg/rpc"
)

func main() {
	var c checker.Connection
	c.ConnectClient("test-rpc")

	// The remote method to be called on the server.
	destinationFail := rpc.Destination{Target: rpc.Service{Name: "server"}, Method: "fail"}

	// Test failure of method on server:
	log.Print("error result test")
	_, err := rpc.Call(c.ClientCtx, destinationFail, time.Time{}, "", nil)
	if err == nil {
		log.Print("test failed")
		os.Exit(1)
	} else {
		log.Print("test succeeded with error: ", err)
	}

	c.CloseClient()
}
