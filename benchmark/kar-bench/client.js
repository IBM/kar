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

const { actor, sys, call } = require('kar-sdk')

// Configuration:

// Time between messages:
const sleepTime = 100 // ms

// Number of timed calls
const numTimedCalls = 100

// Number of warmup calls.
const numDiscardedCalls = 10

if (!process.env.KAR_RUNTIME_PORT) {
  console.error('KAR_RUNTIME_PORT must be set. Aborting.')
  process.exit(1)
}

function sleep (ms) {
  return new Promise(resolve => setTimeout(resolve, ms))
}

async function measureCall (numDiscardedCalls, numTimedCalls) {
  // Result:
  var sumOfAllCalls = 0

  // Variables created once.
  var result

  // Perform requests discarding the first numDiscardedCalls.
  for (let i = 0; i < numDiscardedCalls + numTimedCalls; i++) {
    var start = new Date().getTime()
    result = await call('bench', 'bench-json', { body: 'test' })
    await result
    var callDuation = new Date().getTime() - start

    // Postprocessing.
    if (i >= numDiscardedCalls) {
      sumOfAllCalls += callDuation
      if (numTimedCalls < 50) {
        console.log(`Durations: ${i - numDiscardedCalls}: ${callDuation} ms`)
      }
    }
    await sleep(sleepTime)
  }
  return sumOfAllCalls
}

async function measureOneWayCall (numDiscardedCalls, numTimedCalls) {
  // Results:
  var sumOfAllRequests = 0
  var sumOfAllResponses = 0

  // Create variables once.
  var result
  var remoteStamp, localStamp

  // Perform requests discarding the first numDiscardedCalls.
  for (let i = 0; i < numDiscardedCalls + numTimedCalls; i++) {
    var start = new Date().getTime()
    result = await call('bench', 'bench-json-one-way', { body: 'test' })
    remoteStamp = await result
    localStamp = new Date().getTime()

    // Postprocessing.
    // HTTP2: if enabled then the stamp needs to be extracted from the body
    // explicitly otherwise the time stamp will be in remoteStamp.
    remoteStamp = remoteStamp.body

    var oneWayCall = parseInt(remoteStamp) - start
    var responseCall = localStamp - parseInt(remoteStamp)
    if (i >= numDiscardedCalls) {
      sumOfAllRequests += oneWayCall
      sumOfAllResponses += responseCall
      if (numTimedCalls < 50) {
        console.log(`Durations: ${i - numDiscardedCalls}: ${oneWayCall} ms`)
      }
    }
    await sleep(sleepTime)
  }
  return [sumOfAllRequests, sumOfAllResponses]
}

async function measureActorCall (numDiscardedCalls, numTimedCalls) {
  // Result:
  var sumOfAllCalls = 0

  // Variables created once.
  var response
  var actorClass = actor.proxy('BenchActor', 'TestActor')

  // Perform requests discarding the first numDiscardedCalls.
  for (let i = 0; i < numDiscardedCalls + numTimedCalls; i++) {
    var start = new Date().getTime()
    response = await actor.call(actorClass, 'simpleMethod')
    var callDuation = new Date().getTime() - start

    // Postprocessing.
    if (i >= numDiscardedCalls) {
      sumOfAllCalls += callDuation
      if (numTimedCalls < 50) {
        console.log(`Durations: ${i - numDiscardedCalls}: ${callDuation} ms`)
      }
    }
    await sleep(sleepTime)
  }

  // Remove actor.
  await actor.remove(actorClass)
  return sumOfAllCalls
}

async function measureActorOneWayCall (numDiscardedCalls, numTimedCalls) {
  // Results:
  var sumOfAllRequests = 0
  var sumOfAllResponses = 0

  // Create variables once.
  var remoteStamp, localStamp
  var actorClass = actor.proxy('BenchActor', 'AnotherTestActor')

  // Perform requests discarding the first numDiscardedCalls.
  for (let i = 0; i < numDiscardedCalls + numTimedCalls; i++) {
    var start = new Date().getTime()
    remoteStamp = await actor.call(actorClass, 'timedMethod')
    localStamp = new Date().getTime()

    // Postprocessing.
    var oneWayCall = parseInt(remoteStamp) - start;
    var responseCall = localStamp - parseInt(remoteStamp)
    if (i >= numDiscardedCalls) {
      sumOfAllRequests += oneWayCall
      sumOfAllResponses += responseCall
      if (numTimedCalls < 50) {
        console.log(`Durations: ${i - numDiscardedCalls}: ${oneWayCall} ms`)
      }
    }
    await sleep(sleepTime)
  }

  // Remove actor.
  await actor.remove(actorClass)
  return [sumOfAllRequests, sumOfAllResponses]
}

// main method
async function main () {
  var sumOfAllCalls = await measureCall(numDiscardedCalls, numTimedCalls)
  var averageCallDuration = sumOfAllCalls / numTimedCalls
  console.log(`Average service call duration: ${averageCallDuration} ms`)

  {
    let [sumOfAllRequests, sumOfAllResponses] = await measureOneWayCall(numDiscardedCalls, numTimedCalls)
    const averageRequestDuration = sumOfAllRequests / numTimedCalls
    const averageResponseDuration = sumOfAllResponses / numTimedCalls
    console.log(`Average service request duration: ${averageRequestDuration} ms`)
    console.log(`Average service response duration: ${averageResponseDuration} ms`)
  }

  sumOfAllCalls = await measureActorCall(numDiscardedCalls, numTimedCalls)
  averageCallDuration = sumOfAllCalls / numTimedCalls
  console.log(`Average actor call duration: ${averageCallDuration} ms`)

  {
    let [sumOfAllRequests, sumOfAllResponses] = await measureActorOneWayCall(numDiscardedCalls, numTimedCalls)
    averageRequestDuration = sumOfAllRequests / numTimedCalls
    averageResponseDuration = sumOfAllResponses / numTimedCalls
    console.log(`Average actor request duration: ${averageRequestDuration} ms`)
    console.log(`Average actor response duration: ${averageResponseDuration} ms`)
  }

  await sys.shutdown()
}

// invoke main
main()
