/*
 * Copyright IBM Corporation 2020,2023
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

const { actor, sys } = require('kar-sdk')
const { v4: uuidv4 } = require('uuid')

const NUM_ACCTS = 1000
const ACCTS_PER_TXN = 2
const NUM_TXNS = 200
const CONCURRENCY = 20
const txns = {}; let successCnt = 0

function getRandomInt (min, max) {
  return Math.floor(Math.random() * (max + 1 - min) + min)
}

function getTimeNanoSec () {
  const hrTime = process.hrtime()
  return (hrTime[0] * 1000000000 + hrTime[1])
}

async function warmUp () {
  for (let i = 0; i < NUM_TXNS / CONCURRENCY; i++) {
    const promises = []
    for (let j = 0; j < CONCURRENCY; j++) {
      promises.push(transfer(true))
    }
    await Promise.all(promises)
  }
}

async function transfer (isWarmUp = false) {
  let success = false; const accts = []
  for (let j = 0; ; j++) {
    const a = getRandomInt(1, NUM_ACCTS)
    if (accts.includes('a' + a)) { continue }
    accts.push('a' + a)
    if (accts.length === ACCTS_PER_TXN) { break }
  }
  const amt = getRandomInt(10, 100)
  const txnId = uuidv4()
  const txn = actor.proxy('MoneyTransfer', txnId)
  const operations = []
  for (let i = 0; i < accts.length; i++) {
    const op = ((i % 2 === 0) ? amt : -amt)
    operations.push(op)
  }
  if (!isWarmUp) {
    txns[txnId] = {}
    txns[txnId].startTimer = getTimeNanoSec()
  }
  success = await actor.call(txn, 'startTxn', accts, operations)
  if (!isWarmUp) {
    txns[txnId].endTimer = getTimeNanoSec()
    if (success) { successCnt++ }
  }
}

async function getLatency (totalTime) {
  let totalLatency = 0
  for (const i in txns) {
    totalLatency += (txns[i].endTimer - txns[i].startTimer)
  }
  console.log('Total Latency in ms: ', totalLatency / 1000000 / NUM_TXNS)
  console.log('Throughput :', NUM_TXNS / totalTime * 1000000000)
}

async function initiateTransfer () {
  for (let i = 0; i < NUM_TXNS / CONCURRENCY; i++) {
    await transfer()
  }
}

async function main () {
  await warmUp()
  const strt = getTimeNanoSec()
  const promises = []
  for (let j = 0; j < CONCURRENCY; j++) {
    promises.push(initiateTransfer())
  }
  await Promise.all(promises)
  const end = getTimeNanoSec()
  console.log(successCnt, 'success out of', NUM_TXNS, 'txns.')
  await getLatency(end - strt)
  console.log('Terminating sidecar')
  await sys.shutdown()
}

main()
