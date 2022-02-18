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

package main

import (
	"log"
	"time"

	"github.com/IBM/kar/core/pkg/checker"
	"github.com/IBM/kar/core/pkg/rpc"
)

func main() {
	var c checker.Connection
	c.ConnectClient("test-rpc")

	log.Print("kill server")
	nodes, _ := rpc.GetServiceNodeIDs("server")
	for _, node := range nodes {
		destination := rpc.Destination{Target: rpc.Node{ID: node}, Method: "exit"}
		rpc.Tell(c.ClientCtx, destination, time.Time{}, []byte("goodbye"))
	}

	c.CloseClient()
}
