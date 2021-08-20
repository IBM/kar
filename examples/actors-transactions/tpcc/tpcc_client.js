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
const { actor, sys } = require('kar-sdk')
const { v4: uuidv4 } = require('uuid')
var c = require('./constants.js')
const verbose = process.env.VERBOSE
const cIdRange = [2010, 3000]
const NUM_TXNS =  100
var txns = {id: { startTimer: null, endTimer: null} }
var txnMetadata = {
  'newOrder': {cnt: 0, success: 0, txns: {}},
  'payment': {cnt: 0, success: 0, txns: {}},
  'delivery': {cnt: 0, success: 0},
  'orderStatus': {cnt: 0, success: 0},
  'stockLevel': {cnt: 0, success: 0}
}
async function getRandomInt(min, max) {
  return Math.floor(Math.random() * (max + 1 - min) + min);
}

async function getTimeNanoSec() {
  var hrTime = process.hrtime()
  return (hrTime[0] * 1000000000 + hrTime[1])
}

async function newOrderTxn() {
  const wId = 'w' + await getRandomInt(1, c.NUM_WAREHOUSES)
  const dId = 'd' + await getRandomInt(1, c.NUM_DISTRICTS)
  const cId = 'c' + await getRandomInt(cIdRange[0], cIdRange[1])
  const numItems = await getRandomInt(5,15)

  let orderLines = {}
  for (let i = 0; i < numItems; i++) {
    const itemId = 'i' + await getRandomInt(8191, 100000)
    const supplyWId = await getRandomInt(1, 100) <= 99? wId : 'w' + await getRandomInt(1, c.NUM_WAREHOUSES)
    const quantity = await getRandomInt(1, 10)
    orderLines[i+1] = { itemId: itemId, supplyWId: supplyWId, quantity:quantity }
  }

  var txn = {}
  txn.wId = wId, txn.dId = dId, txn.cId = cId
  txn.olCnt = numItems
  txn.orderLines = orderLines

  var txnId = uuidv4()
  let txnActor = actor.proxy('NewOrderTxn', txnId)
  txnMetadata.newOrder.txns[txnId] = {}
  txnMetadata.newOrder.txns[txnId].startTimer = await getTimeNanoSec()
  const success = await actor.call(txnActor, 'startTxn', txn)
  txnMetadata.newOrder.txns[txnId].endTimer = await getTimeNanoSec()
  if (success) { txnMetadata.newOrder.success++ }
  if (verbose) { console.log("New order txn complete") }
}

async function paymentTxn() {
  const wId = 'w' + await getRandomInt(1, c.NUM_WAREHOUSES)
  const dId = 'd' + await getRandomInt(1, c.NUM_DISTRICTS)
  const cId = 'c' + await getRandomInt(cIdRange[0], cIdRange[1])

  const amount = await getRandomInt(1, 5000)
  const txnId = uuidv4()
  const txnActor = actor.proxy('PaymentTxn', txnId)
  var txn = {}
  txn.wId = wId, txn.dId = dId, txn.cId = cId
  txn.amount = amount
  txnMetadata.payment.txns[txnId] = {}
  txnMetadata.payment.txns[txnId].startTimer = await getTimeNanoSec()
  const success = await actor.call(txnActor, 'startTxn', txn)
  txnMetadata.payment.txns[txnId].endTimer = await getTimeNanoSec()
  if (success) { txnMetadata.payment.success++ }
  if (verbose) { console.log("Payment complete") }
}

async function orderStatusTxn() {
  const wId = 'w' + await getRandomInt(1, c.NUM_WAREHOUSES)
  const dId = 'd' + await getRandomInt(1, c.NUM_DISTRICTS)
  const cId = 'c' + await getRandomInt(cIdRange[0], cIdRange[1])

  const txnActor = actor.proxy('OrderStatusTxn', uuidv4())
  var txn = {}
  txn.wId = wId, txn.dId = dId, txn.cId = cId
  await actor.tell(txnActor, 'startTxn', txn)
  if (verbose) { console.log("Order status txn complete") }
}

async function deliveryTxn() {
  const wId = 'w' + await getRandomInt(1, c.NUM_WAREHOUSES)
  const carrierId = await getRandomInt(1, 10)
  const deliveryDate = new Date()
  const txnActor = actor.proxy('DeliveryTxn', uuidv4())
  var txn = {}
  txn.wId = wId, txn.carrierId = carrierId, txn.deliveryDate = deliveryDate
  await actor.tell(txnActor, 'startTxn', txn)
  if (verbose) { console.log("Delivery txn complete") }
}

async function stockLevelTxn() {
  const wId = 'w' + await getRandomInt(1, c.NUM_WAREHOUSES)
  const dId = 'd' + await getRandomInt(1, c.NUM_DISTRICTS)
  const threshold = await getRandomInt(10, 20)

  const txnActor = actor.proxy('StockLevelTxn', uuidv4())
  var txn = {}
  txn.wId = wId, txn.dId = dId, txn.threshold = threshold
  await actor.tell(txnActor, 'startTxn', txn)
  if (verbose) { console.log("Stock level txn complete") }
}

async function getLatency() {
  var newOrderLatency = 0, paymentLatency = 0
  for(const i in txnMetadata.newOrder.txns) {
    const txn = txnMetadata.newOrder.txns[i]
    newOrderLatency += (txn.endTimer - txn.startTimer)
  }
  for(const i in txnMetadata.payment.txns) {
    const txn = txnMetadata.payment.txns[i]
    paymentLatency += (txn.endTimer - txn.startTimer)
  }
  console.log('New Order Latency in ms: ', newOrderLatency/1000000/txnMetadata.newOrder.cnt)
  console.log('Payment Latency in ms: ', paymentLatency/1000000/txnMetadata.payment.cnt)
}
async function main () {
  for (let i = 0; i < NUM_TXNS; i++) {
    const r = await getRandomInt(1, 100)
    if (r < 44) { await newOrderTxn(); txnMetadata.newOrder.cnt++ }
    else if (r < 88) { await paymentTxn(); txnMetadata.payment.cnt++ }
    else if (r < 92) { await orderStatusTxn(); txnMetadata.orderStatus.cnt++ }
    else if (r < 96) { await deliveryTxn(); txnMetadata.delivery.cnt++ }
    else { await stockLevelTxn(); txnMetadata.stockLevel.cnt++ }
  }

  for ( const i in txnMetadata) {
    console.log('Txn cnt of ', i , 'is', txnMetadata[i].cnt)
  }
  console.log(txnMetadata.payment.success + txnMetadata.newOrder.success, 'out of ',
             (txnMetadata.newOrder.cnt + txnMetadata.payment.cnt), 'successful txns.')
  console.log(txnMetadata.payment.success, 'out of ', (txnMetadata.payment.cnt), 'successful Payment txns.')
  console.log(txnMetadata.newOrder.success, 'out of ', (txnMetadata.newOrder.cnt), 'successful NewOrder txns.')
  await getLatency()
  console.log('Terminating sidecar')
  await sys.shutdown()
}

main()
