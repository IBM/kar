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

package rpctest

import (
	"fmt"
	"testing"

	"github.com/IBM/kar/core/pkg/checker"
)

var (
	server  checker.Check
	testTag string = "[RPC Call]"
)

// Mandatory step: start the server responsible for this test.
func TestStartServer(t *testing.T) {
	fmt.Println(testTag, "Start server.")

	// Pass in the name of the server file you wish to run:
	server.RunServer(t, "server")
}

// ---------------------------------------------------------------------------
// Starting to test Call method
// ---------------------------------------------------------------------------
func CallMethod(t *testing.T) {
	t.Parallel()
	var client checker.Check

	fmt.Println(testTag, "Remote call to increment.")

	// Client checks:
	client.CheckClient("incr test")
	client.CheckClient("result: 43")

	// Run tests
	client.RunClientCheck(t, "test-call-method")
}

func CallDeadlineExpired(t *testing.T) {
	t.Parallel()
	var client checker.Check

	fmt.Println(testTag, "Check expired deadline error.")

	// Client checks:
	client.CheckClient("deadline test")
	client.CheckClient("test succeeded with error: deadline expired")

	// Run tests
	client.RunClientCheck(t, "test-call-deadline-expired")
}

func CallUndefinedMethod(t *testing.T) {
	t.Parallel()
	var client checker.Check

	fmt.Println(testTag, "Check undefined method error.")

	// Client checks:
	client.CheckClient("undefined method test")
	client.CheckClient("test succeeded with error: undefined method foo")

	// Run tests
	client.RunClientCheck(t, "test-call-undefined-method")
}

func CallErrorResult(t *testing.T) {
	t.Parallel()
	var client checker.Check

	fmt.Println(testTag, "Check error result.")

	// Client checks:
	client.CheckClient("error result test")
	client.CheckClient("test succeeded with error: failed")

	// Run tests
	client.RunClientCheck(t, "test-call-error-result")
}

func CallAsync(t *testing.T) {
	t.Parallel()
	var client checker.Check

	fmt.Println(testTag, "Check async.")

	// Client checks:
	client.CheckClient("async test")
	client.CheckClient("async await successful")
	client.CheckClient("async result: 43")

	// Run tests
	client.RunClientCheck(t, "test-call-async")
}

func CallSequential(t *testing.T) {
	t.Parallel()
	var client checker.Check

	fmt.Println(testTag, "Check sequential method calls.")

	// Client checks:
	client.CheckClient("sequential test")
	client.CheckClient("result: 242")

	// Run tests
	client.RunClientCheck(t, "test-call-sequential")
}

func CallSequentialActor(t *testing.T) {
	t.Parallel()
	var client checker.Check

	fmt.Println(testTag, "Check session method call.")

	// Client checks:
	client.CheckClient("sequential actor test")
	client.CheckClient("result: 10")

	// Run tests
	client.RunClientCheck(t, "test-call-sequential-actor")
}

func TestRPCCall(t *testing.T) {
	t.Run("Regular method call", CallMethod)
	t.Run("Deadline expired call", CallDeadlineExpired)
	t.Run("Undefined method call", CallUndefinedMethod)
	t.Run("Error result call", CallErrorResult)
	t.Run("Async call", CallAsync)
	t.Run("Sequential call", CallSequential)
	t.Run("Sequential actor call", CallSequentialActor)
}

// ---------------------------------------------------------------------------
// End of Call tests
// ---------------------------------------------------------------------------

func TestStopServer(t *testing.T) {
	fmt.Println(testTag, "Stop server and check output.")

	// Kill server use the exit client:
	var client checker.Check
	client.CheckClient("kill server")
	client.CheckClient("success")
	client.RunClientCheck(t, "test-exit-client")

	// Server checks:
	server.CheckServer("processing messages")
	server.CheckServer("goodbye")

	// Check server output:
	server.RunServerCheck(t)
}
