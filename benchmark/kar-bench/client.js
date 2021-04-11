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

const { actor, call } = require('kar-sdk')

// Configuration:

// Time between messages:
const sleepTime = 10 // ms

// Number of timed calls
const numTimedCalls = 100

// Number of warmup calls.
const numDiscardedCalls = 10

const stats = {
  serviceEndToEnd: [],
  serviceOneWayRequest: [],
  serviceOneWayResponse: [],
  actorEndToEnd: [],
  actorOneWayRequest: [],
  actorOneWayResponse: []
}

if (!process.env.KAR_RUNTIME_PORT) {
  console.error('KAR_RUNTIME_PORT must be set. Aborting.')
  process.exit(1)
}

function sleep (ms) {
  return new Promise(resolve => setTimeout(resolve, ms))
}

function clearStats () {
  console.log('Cleared statistics')
  stats.actorEndToEnd = []
  stats.actorOneWayRequest = []
  stats.actorOneWayResponse = []
  stats.serviceEndToEnd = []
  stats.serviceOneWayRequest = []
  stats.serviceOneWayResponse = []
}

function reportMetrics (data, tag) {
  const mean = data.reduce((a, b) => a + b, 0) / data.length
  const squareDiffs = data.map(x => (x - mean) * (x - mean))
  const avgSquareDiff = squareDiffs.reduce((a, b) => a + b, 0) / squareDiffs.length
  const stdDev = Math.sqrt(avgSquareDiff)
  console.log(`${tag}: samples = ${data.length}; mean = ${mean.toFixed(3)}; stddev = ${stdDev.toFixed(3)}`)
}

function reportStats () {
  reportMetrics(stats.serviceEndToEnd, 'Service: end-to-end')
  reportMetrics(stats.serviceOneWayRequest, 'Service: one-way-request')
  reportMetrics(stats.serviceOneWayResponse, 'Service: one-way-response')

  reportMetrics(stats.actorEndToEnd, 'Actor: end-to-end')
  reportMetrics(stats.actorOneWayRequest, 'Actor: one-way-request')
  reportMetrics(stats.actorOneWayResponse, 'Actor: one-way-response')
}

async function measureCall (numCalls) {
  // Perform requests
  for (let i = 0; i < numCalls; i++) {
    const start = Date.now()
    const result = await call('bench', 'bench-json', { body: 'test' })
    await result
    stats.serviceEndToEnd.push(Date.now() - start)
    await sleep(sleepTime)
  }
}

async function measureOneWayCall (numCalls) {
  // Perform requests
  for (let i = 0; i < numCalls; i++) {
    var start = Date.now()
    const result = await call('bench', 'bench-json-one-way', { body: 'test' })
    var remoteStamp = await result
    const localStamp = Date.now()

    // Postprocessing.
    // HTTP2: if enabled then the stamp needs to be extracted from the body
    // explicitly otherwise the time stamp will be in remoteStamp.
    remoteStamp = remoteStamp.body

    stats.serviceOneWayRequest.push(parseInt(remoteStamp) - start)
    stats.serviceOneWayResponse.push(localStamp - parseInt(remoteStamp))
    await sleep(sleepTime)
  }
}

async function measureActorCall (numCalls) {
  var actorClass = actor.proxy('BenchActor', 'TestActor')

  // Perform requests
  for (let i = 0; i < numCalls; i++) {
    const start = Date.now()
    const response = await actor.call(actorClass, 'simpleMethod')
    await response
    stats.actorEndToEnd.push(Date.now() - start)
    await sleep(sleepTime)
  }
}

async function measureActorOneWayCall (numCalls) {
  var actorClass = actor.proxy('BenchActor', 'AnotherTestActor')

  // Perform requests
  for (let i = 0; i < numCalls; i++) {
    const start = Date.now()
    const remoteStamp = await actor.call(actorClass, 'timedMethod')
    const localStamp = Date.now()
    stats.actorOneWayRequest.push(parseInt(remoteStamp) - start)
    stats.actorOneWayResponse.push(localStamp - parseInt(remoteStamp))
    await sleep(sleepTime)
  }
}

// main method
async function main () {
  console.log(`Executing ${numDiscardedCalls} warmup operations`)
  await measureCall(numDiscardedCalls)
  await measureOneWayCall(numDiscardedCalls)
  await measureActorCall(numDiscardedCalls)
  await measureActorOneWayCall(numDiscardedCalls)
  reportStats()
  clearStats()

  while (true) {
    await measureCall(numTimedCalls)
    await measureOneWayCall(numTimedCalls)
    await measureActorCall(numTimedCalls)
    await measureActorOneWayCall(numTimedCalls)
    reportStats()
  }
}

// invoke main
main()
