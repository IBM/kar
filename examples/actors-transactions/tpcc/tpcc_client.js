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
const cIdRange = [1, 1]

async function getRandomInt(min, max) {
  min = Math.ceil(min);
  max = Math.floor(max)+1;
  return Math.floor(Math.random() * (max - min) + min); //The maximum is exclusive and the minimum is inclusive
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
    const quantity = await getRandomInt(1, c.NUM_DISTRICTS)
    orderLines[i+1] = { itemId: itemId, supplyWId: supplyWId, quantity:quantity }
  }

  var txn = {}
  txn.wId = wId, txn.dId = dId, txn.cId = cId
  txn.olCnt = numItems
  txn.orderLines = {orderLines: orderLines, v:0}

  const txnActor = actor.proxy('NewOrderTxn', uuidv4())
  await actor.call(txnActor, 'startTxn', txn)
  // const actor1 = actor.proxy('Order', 'w1:d1:o1')
  // const cat = await actor.call(actor1, 'getOrder')
  // console.log(cat.orderLines.orderLines['1'])
  console.log("New order txn complete")
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
  await actor.call(txnActor, 'startTxn', txn)
  console.log("Payment complete")
}

async function orderStatusTxn() {
  const wId = 'w' + await getRandomInt(1, c.NUM_WAREHOUSES)
  const dId = 'd' + await getRandomInt(1, c.NUM_DISTRICTS)
  const cId = 'c' + await getRandomInt(cIdRange[0], cIdRange[1])

  const txnActor = actor.proxy('OrderStatusTxn', uuidv4())
  var txn = {}
  txn.wId = wId, txn.dId = dId, txn.cId = cId
  await actor.call(txnActor, 'startTxn', txn)
  console.log("Order status txn complete")
}

async function deliveryTxn() {
  const wId = 'w' + await getRandomInt(1, c.NUM_WAREHOUSES)
  const carrierId = await getRandomInt(1, 10)
  const deliveryDate = new Date()
  const txnActor = actor.proxy('DeliveryTxn', uuidv4())
  var txn = {}
  txn.wId = wId, txn.carrierId = carrierId, txn.deliveryDate = deliveryDate
  await actor.call(txnActor, 'startTxn', txn)
  console.log("Delivery txn complete")
}

async function main () {
  await newOrderTxn()
  await paymentTxn()
  await orderStatusTxn()
  await deliveryTxn()
  console.log('Terminating sidecar')
  await sys.shutdown()
}

main()
