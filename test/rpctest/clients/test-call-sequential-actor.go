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

	// The remote method to be called on the server:
	destinationIncr := rpc.Destination{Target: rpc.Session{Name: "actor", ID: "instance"}, Method: "incr"}

	// Send requests sequentially to actor method:
	log.Print("sequential actor test")
	n := byte(0)
	for i := 0; i < 1; i++ {
		x, err := rpc.Call(c.ClientCtx, destinationIncr, time.Time{}, []byte{n})
		if err != nil {
			log.Print(err)
			os.Exit(1)
		}
		n = x[0]
	}
	log.Print("result: ", n)

	c.CloseClient()
}