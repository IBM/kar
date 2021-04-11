/*
 * Copyright IBM Corporation 2020,2021
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
  const mean = data.reduce((a, b) => a + b, 0) / data.length
  const squareDiffs = data.map(x => (x - mean) * (x - mean))
  const avgSquareDiff = squareDiffs.reduce((a, b) => a + b, 0) / squareDiffs.length
  const stdDev = Math.sqrt(avgSquareDiff)
  console.log(`${tag}: samples = ${data.length}; mean = ${mean.toFixed(3)}; stddev = ${stdDev.toFixed(3)}`)
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
    var start = Date.now()
    const result = await fetch(url, {
      method: 'POST',
      body: 'Test',
      headers: { 'Content-Type': 'text/plain' }
    })
    await result.text()
    stats.endToEnd.push(Date.now() - start)
    await sleep(sleepTime)
  }
}

async function measureOneWayCall (numCalls, serverIP) {
  const url = call_url('bench-text-one-way', serverIP)
  // Perform requests
  for (let i = 0; i < numCalls; i++) {
    var start = Date.now()
    const result = await fetch(url, {
      method: 'POST',
      body: 'Test',
      headers: { 'Content-Type': 'text/plain' }
    })
    const remoteStamp = await result.text()
    const localStamp = Date.now()

    stats.request.push(parseInt(remoteStamp) - start)
    stats.response.push(localStamp - parseInt(remoteStamp))
    await sleep(sleepTime)
  }
}

// main method
async function main () {
  const serverIP = process.env.SERVER_IP || 'localhost'

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
