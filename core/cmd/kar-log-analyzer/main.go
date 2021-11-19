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
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	karlog     string
	applog     string
	rebalances []rebalanceRecord = []rebalanceRecord{}
	failures   []failureRecord   = []failureRecord{}
)

type rebalanceRecord struct {
	startTime time.Time
	duration  time.Duration
}

type failureRecord struct {
	startTime    time.Time
	maximumOrder time.Duration
	detection    time.Duration
	rebalance    time.Duration
}

func init() {
	flag.StringVar(&karlog, "k", "", "file name of sidecar log to process")
	flag.StringVar(&applog, "a", "", "file name of fault driver log to process")

	flag.CommandLine.Parse(os.Args[1:])
}

func readKarLog() {
	file, err := os.Open(karlog)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	startTime := time.Time{}
	recovering := false

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		msg := strings.Split(line, "[INFO]")
		if len(msg) == 2 {
			ts, err := time.Parse("2006/01/02 15:04:05", strings.TrimSpace(msg[0]))
			if err != nil {
				panic(fmt.Errorf("Can't parse time %v: %v", msg[0], err))
			}
			if !recovering && strings.Contains(msg[1], "completed generation") {
				startTime = ts
				recovering = true
			} else if strings.Contains(msg[1], "processing messages") {
				if recovering {
					outage := ts.Sub(startTime)
					rebalances = append(rebalances, rebalanceRecord{startTime: ts, duration: outage})
					startTime = time.Time{}
					recovering = false
				}
			}
		}
	}
	fmt.Printf("Parsed %v rebalance events\n", len(rebalances))
}

func readAppLog() {
	file, err := os.Open(applog)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	maxOrderLatency := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "k3d node stop") {
			tmp := strings.Split(line, "k3d")[0]
			tmp = strings.Replace(tmp, ", 24:", ", 00:", 1)
			ts, err := time.Parse("01/02/2006, 15:04:05", strings.TrimSpace(tmp))
			if err != nil {
				panic(fmt.Errorf("Can't parse time %v: %v", tmp, err))
			}
			failures = append(failures, failureRecord{startTime: ts})
			if len(failures) > 1 {
				failures[len(failures)-2].maximumOrder = time.Duration(maxOrderLatency) * time.Millisecond
			}
			maxOrderLatency = 0
		} else if strings.Contains(line, "child message:") {
			o := strings.Split(line, "]")
			orderLatency, err := strconv.Atoi(strings.TrimSpace(o[1]))
			if err == nil && orderLatency > maxOrderLatency {
				maxOrderLatency = orderLatency
			}
		}
	}
	failures[len(failures)-1].maximumOrder = time.Duration(maxOrderLatency) * time.Millisecond
	fmt.Printf("Parsed %v failure events\n", len(failures))
}

func correlateLogs() {
	rebalanceIdx := 0
	for i, f := range failures {
		for rebalances[rebalanceIdx].startTime.Before(f.startTime) {
			rebalanceIdx += 1
		}
		r := rebalances[rebalanceIdx]
		// fmt.Printf("Matching: %v %v\n", f, r)
		failures[i].detection = r.startTime.Sub(f.startTime)
		failures[i].rebalance = r.duration
	}
}

func printSummary() {
	fmt.Printf("Start Time, Failure Number, Kafka Detection Threshold, KAR Detection, KAR Recovery, Total Outage, Max Order Latency\n")
	kafkaMin := 10.0
	for i, f := range failures {
		fmt.Printf("%v, %v, %v, %v, %v, %v, %v\n", f.startTime, i+1, kafkaMin, f.detection.Seconds()-kafkaMin,
			f.rebalance.Seconds(), f.detection.Seconds()+f.rebalance.Seconds(), f.maximumOrder.Seconds())
	}
}

func main() {
	fmt.Printf("input kar log: %v\n", karlog)
	fmt.Printf("input app log: %v\n", applog)
	readKarLog()
	readAppLog()
	correlateLogs()
	printSummary()
}
