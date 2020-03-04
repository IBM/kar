package launcher

import (
	"bufio"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.ibm.com/solsa/kar.git/pkg/logger"
)

var wg = sync.WaitGroup{}

// dump adds a time stamp and a prefix to each line of a log
func dump(prefix string, in io.Reader) {
	defer wg.Done()
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		log.Print(prefix, scanner.Text())
	}
}

// Run command with given arguments and environment
func Run(args, env []string) {
	logger.Info("launching service...")
	cmd := exec.Command(args[0], args[1:]...)
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

	wg.Add(1)
	go dump("[STDOUT] ", stdout)
	wg.Add(1)
	go dump("[STDERR] ", stderr)
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		if v, ok := err.(*exec.ExitError); ok {
			logger.Info("service exited with status code %d", v.ExitCode())
		} else {
			logger.Fatal("error waiting for service: %v", err)
		}
	} else {
		logger.Info("service exited normally")
	}
}
