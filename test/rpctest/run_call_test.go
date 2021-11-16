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
	"testing"
	"os/exec"
	"regexp"
	"bytes"
	"bufio"
	"strings"
	"io"
	"fmt"

	"github.com/IBM/kar/core/pkg/checker"
)

// TestIncrement -- test the remote call to the increment function
func TestIncrement(t *testing.T) {
	fmt.Println("Check increment.")
	var c checker.Check

	// Client checks:
	c.checkClient("incr test")
	c.checkClient("result: 43")
	c.checkClient("kill server")
	c.checkClient("success")

	// Server checks:
	c.checkServer("processing messages")
	c.checkServer("goodbye")

	// Run tests
	c.runCheck(t, "test-call")
}