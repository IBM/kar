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

	// Send request to server to increment passed in value of 42.
	log.Print("incr test")
	result, err := rpc.Call(c.ClientCtx, destinationIncr, time.Time{}, []byte{42})
	if err != nil {
		log.Print("incr test failed")
		os.Exit(1)
	} else {
		log.Print("result: ", result[0])
	}

	c.CloseClient()

	// // Send a request with expired deadline, expect error `deadline expired`
	// log.Print("deadline test")
	// _, err = rpc.Call(c.clientCtx, destinationIncr, time.Now().Add(-time.Hour), []byte{42})
	// if err == nil {
	// 	log.Print("test failed")
	// 	os.Exit(1)
	// } else {
	// 	log.Print("test succeeded with error: ", err)
	// }

	// log.Print("undefined method test")
	// _, err = rpc.Call(ctx, rpc.Service{Name: "server"}, "foo", time.Time{}, nil)
	// if err == nil {
	// 	log.Print("test failed")
	// 	os.Exit(1)
	// }
	// log.Print("error: ", err)

	// log.Print("error result test")
	// _, err = rpc.Call(ctx, rpc.Service{Name: "server"}, "fail", time.Time{}, nil)
	// if err == nil {
	// 	log.Print("test failed")
	// 	os.Exit(1)
	// }
	// log.Print("error: ", err)

	// log.Print("async test")
	// _, rp, err := rpc.Async(ctx, rpc.Service{Name: "server"}, "incr", time.Time{}, []byte{42})
	// if err != nil {
	// 	log.Print(err)
	// 	os.Exit(1)
	// }
	// response := <-rp
	// if response.Err != nil {
	// 	log.Print("async await test failed")
	// 	os.Exit(1)
	// }
	// n := response.Value[0]
	// if n != 43 {
	// 	log.Print("async await test failed")
	// 	os.Exit(1)
	// }

	// log.Print("sequential test")
	// n = byte(42)
	// for i := 0; i < 200; i++ {
	// 	x, err := rpc.Call(ctx, rpc.Service{Name: "server"}, "incr", time.Time{}, []byte{n})
	// 	if err != nil {
	// 		log.Print(err)
	// 		os.Exit(1)
	// 	}
	// 	n = x[0]
	// }

	// if n != 242 {
	// 	log.Print("sequential test failed")
	// 	os.Exit(1)
	// }

	// log.Print("sequential actor test")
	// n = byte(0)
	// for i := 0; i < 10; i++ {
	// 	x, err := rpc.Call(ctx, rpc.Session{Name: "actor", ID: "instance"}, "incr", time.Time{}, []byte{n})
	// 	if err != nil {
	// 		log.Print(err)
	// 		os.Exit(1)
	// 	}
	// 	n = x[0]
	// }

	// if n != 10 {
	// 	log.Print("sequential stor test failed")
	// 	os.Exit(1)
	// }

	// log.Print("parallel test")
	// ch := make(chan byte)
	// for i := byte(0); i < 200; i++ {
	// 	x := i
	// 	go func() {
	// 		y, err := rpc.Call(ctx, rpc.Service{Name: "server"}, "incr", time.Time{}, []byte{x + 42})
	// 		if err != nil {
	// 			ch <- 0

	// 		} else {
	// 			ch <- y[0]
	// 		}
	// 	}()
	// }
	// t := 0
	// for i := 0; i < 200; i++ {
	// 	t = t + int(<-ch)
	// }
	// if t != 28500 {
	// 	log.Print("parallel test failed")
	// 	os.Exit(1)
	// }

	// log.Print("kill server")
	// nodes, _ := rpc.GetServiceNodeIDs("server")
	// for _, node := range nodes {
	// 	destination := rpc.Destination{Target: rpc.Node{ID: node}, Method: "exit"} 
	// 	rpc.Tell(ctx, destination, time.Time{}, []byte("goodbye"))
	// }

	// log.Print("success")

	// cancel()
	// <-closed
}