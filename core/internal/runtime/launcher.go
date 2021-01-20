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

package runtime

import (
	"bufio"
	"context"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/IBM/kar.git/core/pkg/logger"
)

// dump adds a time stamp and a prefix to each line of a log
func dump(prefix string, in io.Reader) {
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		log.Print(prefix, scanner.Text())
	}
}

// Run command with given arguments and environment
func Run(ctx context.Context, args, env []string) (exitCode int) {
	logger.Info("launching service...")
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.Fatal("failed to capture stdout from service: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		logger.Fatal("failed to capture stderr from service: %v", err)
	}
	if err := cmd.Start(); err != nil {
		logger.Fatal("failed to start service: %v", err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		dump("[STDOUT] ", stdout)
	}()
	dump("[STDERR] ", stderr)
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		if v, ok := err.(*exec.ExitError); ok {
			if v.ExitCode() == -1 {
				logger.Info("service was interrupted")
			} else {
				logger.Info("service exited with status code %d", v.ExitCode())
				exitCode = v.ExitCode()
			}
		} else {
			logger.Error("error waiting for service: %v", err)
		}
	} else {
		logger.Info("service exited normally")
	}
	return
}
