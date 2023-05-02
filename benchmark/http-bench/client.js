/*
 * Copyright IBM Corporation 2020,2022
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// retry http requests up to 10 times over 10s
const fetch = require('fetch-retry')(require('node-fetch'), { retries: 10 })

// Time between messages:
const sleepTime = 10 // ms

// Number of timed calls
const numTimedCalls = 100

// Number of warmup calls.
const numDiscardedCalls = 10

const stats = {
  endToEnd: [],
  request: [],
  response: []
}

function clearStats () {
  console.log('Cleared statistics')
  stats.endToEnd = []
  stats.request = []
  stats.response = []
}

function reportMetrics (data, tag) {
  data.sort((a, b) => a - b);
  const mean = data.reduce((a, b) => a + b, 0) / data.length
  const squareDiffs = data.map(x => (x - mean) * (x - mean))
  const avgSquareDiff = squareDiffs.reduce((a, b) => a + b, 0) / squareDiffs.length
  const stdDev = Math.sqrt(avgSquareDiff)
  const median = data[data.length / 2]
  const nine = data[data.length * 9 / 10]
  const nineNine = data[data.length * 99 / 100]
  console.log(`${tag}: samples = ${data.length}; mean = ${mean.toFixed(3)}; median = ${median}; 90th = ${nine}; 99th= ${nineNine}; stddev = ${stdDev.toFixed(3)}`)
}

function reportStats () {
  reportMetrics(stats.endToEnd, 'HTTP: end-to-end')
  reportMetrics(stats.request, 'HTTP: one-way-request')
  reportMetrics(stats.response, 'HTTP: one-way-response')
}

// request url for a given server:port and route on that server
function call_url (route, serverIP) {
  return `http://${serverIP}:9000/${route}`
}

function sleep (ms) {
  return new Promise(resolve => setTimeout(resolve, ms))
}

async function measureCall (numCalls, serverIP) {
  const url = call_url('bench-text', serverIP)
  // Perform requests.
  for (let i = 0; i < numCalls; i++) {
    const start = process.hrtime.bigint()
    const result = await fetch(url, {
      method: 'POST',
      body: 'Test',
      headers: { 'Content-Type': 'text/plain' }
    })
    await result.text()
    const end = process.hrtime.bigint()
    stats.endToEnd.push(Number(end - start) / 1e6)
    await sleep(sleepTime)
  }
}

async function measureOneWayCall (numCalls, serverIP) {
  const url = call_url('bench-text-one-way', serverIP)
  // Perform requests
  for (let i = 0; i < numCalls; i++) {
    var start = process.hrtime.bigint()
    const result = await fetch(url, {
      method: 'POST',
      body: 'Test',
      headers: { 'Content-Type': 'text/plain' }
    })
    const remoteStamp = await result.text()
    const end = process.hrtime.bigint()
    const midTime = BigInt(remoteStamp)

    stats.request.push(Number(midTime - start) / 1e6)
    stats.response.push(Number(end - midTime) / 1e6)
    await sleep(sleepTime)
  }
}

// main method
async function main () {
  const serverIP = process.env.SERVER_IP || '127.0.0.1'

  console.log(`Executing ${numDiscardedCalls} warmup operations`)
  await measureCall(numDiscardedCalls, serverIP)
  await measureOneWayCall(numDiscardedCalls, serverIP)
  reportStats()
  clearStats()

  while (true) {
    await measureCall(numTimedCalls, serverIP)
    await measureOneWayCall(numTimedCalls, serverIP)
    reportStats()
  }
}

// invoke main
main()
