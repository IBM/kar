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

async function getRandomInt(min, max) {
  min = Math.ceil(min);
  max = Math.floor(max)+1;
  return Math.floor(Math.random() * (max - min) + min); //The maximum is exclusive and the minimum is inclusive
}

async function newOrderTxn() {
  const wId = 'w' + await getRandomInt(1, c.NUM_WAREHOUSES)
  const dId = 'd' + await getRandomInt(1, 10)
  const cId = 'c' + await getRandomInt(1023, 3000)
  const numItems = await getRandomInt(5,15)

  let orderLines = {}
  for (let i = 0; i < numItems; i++) {
    const itemId = 'i' + await getRandomInt(8191, 100000)
    const supplyWId = await getRandomInt(1, 100) >= 99? wId : await getRandomInt(1, c.NUM_WAREHOUSES)
    const quantity = await getRandomInt(1, 10)
    orderLines[i+1] = { itemId: itemId, supplyWId: supplyWId, quantity:quantity }
  }

  var txn = {}
  txn.wId = wId, txn.dId = dId, txn.cId = cId
  txn.olCnt = numItems
  txn.orderLines = orderLines

  console.log(txn)
  const txnActor1 = actor.proxy('NewOrderTxn', uuidv4())
  await actor.call(txnActor1, 'startTxn', txn)
}

async function paymentTxn() {
  const wId = 'w' + await getRandomInt(1, c.NUM_WAREHOUSES)
  const dId = 'd' + await getRandomInt(1, 10)
  const cId = 'c' + await getRandomInt(1023, 3000)

  const amount = await getRandomInt(1, 5000)
  const txnActor = actor.proxy('PaymentTxn', uuidv4())
  var txn = {}
  txn.wId = wId, txn.dId = dId, txn.cId = cId
  txn.amount = amount
  await actor.call(txnActor, 'startTxn', txn)
}

async function main () {
  await newOrderTxn()
  await paymentTxn()
  console.log('Terminating sidecar')
  await sys.shutdown()
}

main()
