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

const e = require('express')
const math = require('mathjs')
const { actor, sys } = require('kar-sdk')
const { v4: uuidv4 } = require('uuid')
var c = require('./constants.js')
const verbose = process.env.VERBOSE
const cIdRange = [2010, 3000]
const WARN_UP_TXNS = 10
const NUM_TXNS =  200
const CONCURRENCY = 20

var txnMetadata = {
  'newOrder': {cnt: 0, success: 0, txns: {}},
  'payment': {cnt: 0, success: 0, txns: {}},
  'delivery': {cnt: 0, success: 0, txns: {}},
  'orderStatus': {cnt: 0, success: 0, txns: {}},
  'stockLevel': {cnt: 0, success: 0, txns: {}}
}

function getRandomInt(min, max) {
  return Math.floor(Math.random() * (max + 1 - min) + min);
}

function getTimeNanoSec() {
  var hrTime = process.hrtime()
  return (hrTime[0] * 1000000000 + hrTime[1])
}

async function getKafkaRedisLatencies() {
  const wId = 'w100'
  const warehouse = actor.proxy('Warehouse', wId)
  await actor.call(warehouse, 'returnNull')
  console.time(`Invocation`);
  await actor.call(warehouse, 'returnNull')
  console.timeEnd(`Invocation`);
  console.time(`RedisReadAll`);
  await actor.call(warehouse, 'getAll')
  console.timeEnd(`RedisReadAll`);
  console.time(`RedisReadPutMultiple`);
  await actor.call(warehouse, 'putMultiple', {a:1, b:2})
  console.timeEnd(`RedisReadPutMultiple`);
  await actor.remove(warehouse)
}

async function warmUp() {
  for (let i = 0; i < WARN_UP_TXNS; i++) {
    const r = getRandomInt(1, 100)
    if (r < 44) { await newOrderTxn(true)}
    else if (r < 88) { await paymentTxn(true)}
    else if (r < 92) { await orderStatusTxn(true)}
    else if (r < 96) { await deliveryTxn(true)}
    else { await stockLevelTxn(true)}
  }
}

async function newOrderTxn(isWarmUp = false) {
  const wId = 'w' + getRandomInt(1, c.NUM_WAREHOUSES)
  const dId = 'd' + getRandomInt(1, c.NUM_DISTRICTS)
  const cId = 'c' + getRandomInt(cIdRange[0], cIdRange[1])
  const numItems = getRandomInt(5,15)

  let orderLines = {}
  for (let i = 0; i < numItems; i++) {
    const itemId = 'i' + getRandomInt(8191, 100000)
    const supplyWId = getRandomInt(1, 100) <= 99? wId : 'w' + getRandomInt(1, c.NUM_WAREHOUSES)
    const quantity = getRandomInt(1, 10)
    orderLines[i+1] = { itemId: itemId, supplyWId: supplyWId, quantity:quantity }
  }
  var txn = {}
  txn.wId = wId, txn.dId = dId, txn.cId = cId
  txn.olCnt = numItems
  txn.orderLines = orderLines

  var txnId = uuidv4()
  let txnActor = actor.proxy('NewOrderTxn', txnId)
  if (!isWarmUp) {
    txnMetadata.newOrder.txns[txnId] = {}
    txnMetadata.newOrder.txns[txnId].startTimer = getTimeNanoSec()
  }
  const success = await actor.call(txnActor, 'startTxn', txn)
  if (!isWarmUp) { 
    txnMetadata.newOrder.txns[txnId].endTimer = getTimeNanoSec()
    if (success) { txnMetadata.newOrder.success++ }
  }
  if (verbose) { console.log("New order txn complete") }
}

async function paymentTxn(isWarmUp = false) {
  const wId = 'w' + getRandomInt(1, c.NUM_WAREHOUSES)
  const dId = 'd' + getRandomInt(1, c.NUM_DISTRICTS)
  const cId = 'c' + getRandomInt(cIdRange[0], cIdRange[1])

  const amount = getRandomInt(1, 5000)
  const txnId = uuidv4()
  const txnActor = actor.proxy('PaymentTxn', txnId)
  var txn = {}
  txn.wId = wId, txn.dId = dId, txn.cId = cId
  txn.amount = amount
  if (!isWarmUp) {
    txnMetadata.payment.txns[txnId] = {}
    txnMetadata.payment.txns[txnId].startTimer = getTimeNanoSec() 
  }
  const success = await actor.call(txnActor, 'startTxn', txn)
  if (!isWarmUp) {
    txnMetadata.payment.txns[txnId].endTimer = getTimeNanoSec()
    if (success) { txnMetadata.payment.success++ }
  }
  if (verbose) { console.log("Payment complete") }
}

async function orderStatusTxn(isWarmUp = false) {
  const wId = 'w' + getRandomInt(1, c.NUM_WAREHOUSES)
  const dId = 'd' + getRandomInt(1, c.NUM_DISTRICTS)
  const cId = 'c' + getRandomInt(cIdRange[0], cIdRange[1])

  const txnId = uuidv4()
  const txnActor = actor.proxy('OrderStatusTxn', uuidv4())
  var txn = {}
  txn.wId = wId, txn.dId = dId, txn.cId = cId
  if (!isWarmUp) {
    txnMetadata.orderStatus.txns[txnId] = {}
    txnMetadata.orderStatus.txns[txnId].startTimer = getTimeNanoSec()
  }
  const status = await actor.call(txnActor, 'startTxn', txn)
  if (!isWarmUp) {
    txnMetadata.orderStatus.txns[txnId].endTimer = getTimeNanoSec()
    if (status.decision) { txnMetadata.orderStatus.success++ }
  }
  if (verbose) { console.log("Order status txn complete") }
}

async function deliveryTxn(isWarmUp = false) {
  const wId = 'w' + getRandomInt(1, c.NUM_WAREHOUSES)
  const carrierId = getRandomInt(1, 10)
  const deliveryDate = new Date()
  const txnId = uuidv4()
  const txnActor = actor.proxy('DeliveryTxn', txnId)
  var txn = {}
  txn.wId = wId, txn.carrierId = carrierId, txn.deliveryDate = deliveryDate
  if (!isWarmUp) {
    txnMetadata.delivery.txns[txnId] = {}
    txnMetadata.delivery.txns[txnId].startTimer = getTimeNanoSec()
  }
  const success = await actor.call(txnActor, 'startTxn', txn)
  if (!isWarmUp) { 
    txnMetadata.delivery.txns[txnId].endTimer = getTimeNanoSec()
    if (success) { txnMetadata.delivery.success++ } }
  if (verbose) { console.log("Delivery txn complete") }
}

async function stockLevelTxn(isWarmUp = false) {
  const wId = 'w' + getRandomInt(1, c.NUM_WAREHOUSES)
  const dId = 'd' + getRandomInt(1, c.NUM_DISTRICTS)
  const threshold = getRandomInt(10, 20)

  const txnId = uuidv4()
  const txnActor = actor.proxy('StockLevelTxn', txnId)
  var txn = {}
  txn.wId = wId, txn.dId = dId, txn.threshold = threshold
  if (!isWarmUp) {
    txnMetadata.stockLevel.txns[txnId] = {}
    txnMetadata.stockLevel.txns[txnId].startTimer = getTimeNanoSec()
  }
  await actor.call(txnActor, 'startTxn', txn)
  if (!isWarmUp) { 
    txnMetadata.stockLevel.txns[txnId].endTimer = getTimeNanoSec()
    txnMetadata.stockLevel.success++ }
  if (verbose) { console.log("Stock level txn complete") }
}

async function getLatency(totalTime) {
  var totalLatency = 0, totalCnt = 0
  for(const i in txnMetadata) {
    for (const j in txnMetadata[i].txns) {
      const txn = txnMetadata[i].txns[j]
      totalLatency += (txn.endTimer - txn.startTimer)
    }
    totalCnt += txnMetadata[i].cnt
  }
  console.log('Total Latency in ms: ', totalLatency/1000000/totalCnt)
  console.log('Throughput :',  totalCnt/totalTime*1000000000)
}

async function main () {
  await warmUp()
  await getKafkaRedisLatencies()
  const strt = getTimeNanoSec()
  for (let i = 0; i < NUM_TXNS/CONCURRENCY; i++) {
    const r = getRandomInt(1, 100)
    let promises = []
    for (let j = 0; j < CONCURRENCY; j++) {
      const r = getRandomInt(1, 100)
      if (r < 44) { promises.push(newOrderTxn()); txnMetadata.newOrder.cnt++ }
      else if (r < 88) { promises.push(paymentTxn()); txnMetadata.payment.cnt++ }
      else if (r < 92) { promises.push(orderStatusTxn()); txnMetadata.orderStatus.cnt++ }
      else if (r < 96) { promises.push(deliveryTxn()); txnMetadata.delivery.cnt++ }
      else { promises.push(stockLevelTxn()); txnMetadata.stockLevel.cnt++ }
    }
    await Promise.all(promises)
  }
  const end = getTimeNanoSec()
  var successTotal = 0
  for (const i in txnMetadata) {
    console.log('Success cnt of', i , 'txn is', txnMetadata[i].success, ' out of ', txnMetadata[i].cnt)
    successTotal += txnMetadata[i].success
  }
  console.log(successTotal, 'success out of', NUM_TXNS, 'txns.')
  await getLatency(end-strt)
  console.log('Terminating sidecar')
  await sys.shutdown()
}

main()
