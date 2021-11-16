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
	"testing"
	"os/exec"
	"regexp"
	"bytes"
	"bufio"
	"strings"
	"io"
)

// Check type
type Check struct {  
	checkClientOrdered []string
	checkServerOrdered []string
}

func (c *Check) checkClient(checkString string) {
	c.checkClientOrdered = append(c.checkClientOrdered, checkString)
}

func (c *Check) checkServer(checkString string) {
	c.checkServerOrdered = append(c.checkServerOrdered, checkString)
}

func (c *Check) runOrderedCheck(t *testing.T, standardOut io.Reader, fileChecks []string) {
	scanner := bufio.NewScanner(standardOut)
	checkIndex := 0
	savedClientOutput := []string{}
	for scanner.Scan() {
		outputLine := scanner.Text()
		savedClientOutput = append(savedClientOutput, outputLine)
		if checkIndex < len(fileChecks) && strings.Contains(outputLine, fileChecks[checkIndex]) {
			checkIndex++
		}
	}
	clientOutput := strings.Join(savedClientOutput, "\n")
	if checkIndex < len(fileChecks) {
		t.Fatalf("Client output error: %s not found in: \n%s\n", fileChecks[checkIndex], clientOutput)
	}
}

func (c *Check) runCheck(t *testing.T, testDirectoryName string) {
	// Run server:
	server := exec.Command("go", "run", testDirectoryName+"/server.go")
	serverOutput, err := server.StderrPipe()
	if err != nil {
		t.Fatalf(`Error running stdout server pipe %v`, err)
	}
	err = server.Start()
	if err != nil {
		t.Fatalf(`Error running server %v`, err)
	}

	// Run client:
	client := exec.Command("go", "run", testDirectoryName+"/client.go")
	clientOutput, err := client.StderrPipe()
	if err != nil {
		t.Fatalf(`Error running stdout server pipe %v`, err)
	}
	err = client.Start()
	if err != nil {
		t.Fatalf(`Error running client %v`, err)
	}

	// Perform checks on the client side:
	c.runOrderedCheck(t, clientOutput, c.checkClientOrdered)

	// Perform checks on the server side:
	c.runOrderedCheck(t, serverOutput, c.checkServerOrdered)

	// Wait for subprocesses to finish:
	client.Wait()
	server.Wait()	
}

func check(t *testing.T, matchString string, rawOutput bytes.Buffer) {
	expected := regexp.MustCompile(`\b`+matchString+`\b`)
	if !expected.MatchString(rawOutput.String()) {
        t.Fatalf(`Expected string %s not found`, matchString)
    }
}
