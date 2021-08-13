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

const { actor, sys } = require('kar-sdk')
const { v4: uuidv4 } = require('uuid')
var c = require('./constants.js')
const verbose = process.env.VERBOSE
const cIdRange = [1, 1]
const NUM_TXNS =  100
var successCnt = 0

async function getRandomInt(min, max) {
  return Math.floor(Math.random() * (max + 1 - min) + min);
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
  txn.orderLines = {val: orderLines, ts:0}

  let txnActor = actor.proxy('NewOrderTxn', uuidv4())
  const success = await actor.call(txnActor, 'startTxn', txn)
  if (success) { successCnt++ }
  if (verbose) { console.log("New order txn complete") }
}

async function paymentTxn() {
  const wId = 'w' + await getRandomInt(1, c.NUM_WAREHOUSES)
  const dId = 'd' + await getRandomInt(1, c.NUM_DISTRICTS)
  const cId = 'c' + await getRandomInt(cIdRange[0], cIdRange[1])

  const amount = await getRandomInt(1, 5000)
  const txnActor = actor.proxy('PaymentTxn', uuidv4())
  var txn = {}
  txn.wId = wId, txn.dId = dId, txn.cId = cId
  txn.amount = amount
  const success = await actor.call(txnActor, 'startTxn', txn)
  if (success) { successCnt++ }
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

async function main () {
  let txnsCnt = [0, 0, 0, 0, 0]
  for (let i = 0; i < NUM_TXNS; i++) {
    const r = await getRandomInt(1, 100)
    if (r < 44) { await newOrderTxn(); txnsCnt[0]++ }
    else if (r < 88) { await paymentTxn(); txnsCnt[1]++ }
    else if (r < 92) { await orderStatusTxn(); txnsCnt[2]++ }
    else if (r < 96) { await deliveryTxn(); txnsCnt[3]++ }
    else { await stockLevelTxn(); txnsCnt[4]++ }
  }
  console.log(txnsCnt)
  console.log(successCnt, 'out of ', (txnsCnt[0]+txnsCnt[1]), 'successful txns.')
  console.log('Terminating sidecar')
  await sys.shutdown()
}

main()
