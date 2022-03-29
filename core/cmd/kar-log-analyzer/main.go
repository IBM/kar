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

// This program is used to correlate events in the log files from
// the k3d fault-driver, kar leader, and kafka logs collected during
// a failure scenario run to enable analysis of performance during failures.
//
// From the fault-driver log, it extracts the the start time of
// each failure and the maximum order latency during the failure.
// From the kafka log, it extracts the start and end time of Kafka rebalances.
// From the kar log, it extracts the end time of the KAR recovery.
//
// Run the experiment using the scripts found in kar-apps/reefer/scripts.
// Collect the kakfa log, kar sidecar log from the leader (will be
// either the monitor or simulator sidecar), and the fault driver log.

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
	kafkalog     string
	failures     []failureEvent   = []failureEvent{}
	rebalances   []rebalanceEvent = []rebalanceEvent{}
	processing   []time.Time      = []time.Time{}
	summaries    []summary        = []summary{}
	failureHisto []int            = []int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
)

type rebalanceEvent struct {
	startTime time.Time
	endTime   time.Time
}

type failureEvent struct {
	startTime    time.Time
	maximumOrder time.Duration
}

type summary struct {
	startTime       time.Time
	totalDuration   time.Duration
	maximumOrder    time.Duration
	detection       time.Duration
	consensus       time.Duration
	reconcilliation time.Duration
	failureCount    int
}

func init() {
	flag.StringVar(&kafkalog, "kafka", "", "file name of kafka log to process")
	flag.StringVar(&karlog, "kar", "", "file name of sidecar log to process")
	flag.StringVar(&applog, "app", "", "file name of fault driver log to process")

	flag.CommandLine.Parse(os.Args[1:])
}

func readKarLog() {
	file, err := os.Open(karlog)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		msg := strings.Split(line, "[INFO]")
		if len(msg) == 2 {
			ts, err := time.Parse("2006/01/02 15:04:05", strings.TrimSpace(msg[0]))
			if err != nil {
				panic(fmt.Errorf("Can't parse time %v: %v", msg[0], err))
			}
			if strings.Contains(msg[1], "processing messages") {
				processing = append(processing, ts)
			}
		}
	}
	fmt.Printf("Found %v processing events\n", len(rebalances))
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
			ts, err := time.Parse("1/2/2006, 15:04:05", strings.TrimSpace(tmp))
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

func readKafkaLog() {
	file, err := os.Open(kafkalog)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	prepare := time.Time{}
	for scanner.Scan() {
		line := scanner.Text()
		msg := strings.Split(line, " INFO ")
		if len(msg) == 2 {
			ts, err := time.Parse("[2006-01-02 15:04:05]", strings.TrimSpace(msg[0]))
			if err != nil {
				panic(fmt.Errorf("Can't parse time %v: %v", msg[0], err))
			}
			if strings.Contains(msg[1], "Preparing to rebalance group") {
				prepare = ts
			} else if strings.Contains(msg[1], "Stabilized group") {
				rebalances = append(rebalances, rebalanceEvent{startTime: prepare, endTime: ts})
				prepare = time.Time{}
			}
		}
	}
	fmt.Printf("Parsed %v kafka rebalances\n", len(rebalances))
}

func correlateLogs() {
	failureIdx := 0
	rebalanceIdx := 0
	processingIdx := 0
	for ; failureIdx < len(failures); failureIdx += 1 {
		pf := failures[failureIdx]
		for rebalances[rebalanceIdx].startTime.Before(pf.startTime) {
			rebalanceIdx += 1
		}
		r := rebalances[rebalanceIdx]
		for processing[processingIdx].Before(r.endTime) {
			processingIdx += 1
		}
		p := processing[processingIdx]
		maxOrder := pf.maximumOrder
		numFailures := 1
		for ; failureIdx+1 < len(failures) && p.After(failures[failureIdx+1].startTime); failureIdx += 1 {
			sf := failures[failureIdx+1]
			// fmt.Printf("Merging failure: %v and %v due to processing %v\n", pf, sf, p)
			numFailures += 1
			if sf.maximumOrder > maxOrder {
				maxOrder = sf.maximumOrder
			}
		}

		outage := p.Sub(pf.startTime)
		detect := r.startTime.Sub(pf.startTime)
		rebalance := r.endTime.Sub(r.startTime)
		reconcile := p.Sub(r.endTime)
		summaries = append(summaries, summary{startTime: pf.startTime, failureCount: numFailures, totalDuration: outage, detection: detect,
			consensus: rebalance, reconcilliation: reconcile, maximumOrder: maxOrder})
		failureHisto[numFailures] += 1
	}
	fmt.Printf("Count of failure clusters: %v\n", failureHisto[1:])
}

func printSummary() {
	fmt.Printf("Start Time, Failure Number, Num Failures, Detection, Consensus, Reconcilliation, Total Outage, Max Order Latency\n")
	for i, f := range summaries {
		fmt.Printf("%v, %v, %v, %.6v, %.6v, %.6v, %.6v, %.6v\n", f.startTime, i+1, f.failureCount, f.detection.Seconds(), f.consensus.Seconds(),
			f.reconcilliation.Seconds(), f.totalDuration.Seconds(), f.maximumOrder.Seconds())
	}
}

func main() {
	fmt.Printf("input kafka log: %v\n", kafkalog)
	fmt.Printf("input kar log: %v\n", karlog)
	fmt.Printf("input app log: %v\n", applog)
	readKafkaLog()
	readKarLog()
	readAppLog()
	correlateLogs()
	printSummary()
}
