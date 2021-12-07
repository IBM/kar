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
	karlog       string
	applog       string
	rebalances   []rebalanceEvent = []rebalanceEvent{}
	failures     []failureEvent   = []failureEvent{}
	summaries    []summary        = []summary{}
	failureHisto []int            = []int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
)

type rebalanceEvent struct {
	startTime time.Time
	duration  time.Duration
}

type failureEvent struct {
	startTime    time.Time
	maximumOrder time.Duration
}

type summary struct {
	startTime          time.Time
	totalDuration      time.Duration
	maximumOrder       time.Duration
	kafkaDetection     time.Duration
	karDetection       time.Duration
	karReconcilliation time.Duration
	failureCount       int
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
					rebalances = append(rebalances, rebalanceEvent{startTime: ts, duration: outage})
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
			failures = append(failures, failureEvent{startTime: ts})
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
	kafkaDetect := 10 * time.Second // for now, assuming a fixed minimum.  Eventually we should analyze the kafka logs and get a real number.
	rebalanceIdx := 0
	failureIdx := 0
	for ; failureIdx < len(failures); failureIdx += 1 {
		pf := failures[failureIdx]
		for rebalances[rebalanceIdx].startTime.Before(pf.startTime) {
			rebalanceIdx += 1
		}
		r := rebalances[rebalanceIdx]
		maxOrder := pf.maximumOrder
		numFailures := 1
		for ; failureIdx+1 < len(failures) && (r.startTime.Add(r.duration)).After(failures[failureIdx+1].startTime); failureIdx += 1 {
			sf := failures[failureIdx+1]
			// fmt.Printf("Merging failure: %v and %v due to rebalance %v\n", pf, sf, r)
			numFailures += 1
			if sf.maximumOrder > maxOrder {
				maxOrder = sf.maximumOrder
			}
		}

		outage := r.startTime.Sub(pf.startTime) + r.duration
		detect := r.startTime.Sub(pf.startTime)
		summaries = append(summaries, summary{startTime: pf.startTime, failureCount: numFailures, totalDuration: outage, kafkaDetection: kafkaDetect,
			karDetection: detect - kafkaDetect, karReconcilliation: r.duration, maximumOrder: maxOrder})
		failureHisto[numFailures] += 1
	}
	fmt.Printf("Count of failure clusters: %v\n", failureHisto[1:])
}

func printSummary() {
	fmt.Printf("Start Time, Failure Number, Num Failures, Total Outage, Kafka Detection, KAR Detection, KAR Reconcilliation, Max Order Latency\n")
	for i, f := range summaries {
		fmt.Printf("%v, %v, %v, %.6v, %.6v, %.6v, %.6v, %.6v\n", f.startTime, i+1, f.failureCount, f.totalDuration.Seconds(), f.kafkaDetection.Seconds(), f.karDetection.Seconds(),
			f.karReconcilliation.Seconds(), f.maximumOrder.Seconds())
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
