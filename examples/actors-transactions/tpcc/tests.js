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
const wId = 'w1'
const dId = 'd1'
const cId = 'c1'

async function getRandomInt(min, max) {
  return Math.floor(Math.random() * (max + 1 - min) + min);
}

async function newOrderTxn(wId, dId, cId, itemIds, quantity) {
  let orderLines = {}
  for (let i in itemIds) {
    const itemId = itemIds[i]
    orderLines[i+1] = { itemId: itemId, supplyWId: wId, quantity:quantity }
  }

  var txn = {}
  txn.wId = wId, txn.dId = dId, txn.cId = cId
  txn.olCnt = itemIds.length
  txn.orderLines = {val: orderLines, ts:0}

  let txnActor = actor.proxy('NewOrderTxn', uuidv4())
  const success = await actor.call(txnActor, 'startTxn', txn)
  console.log('Transaction success status: ', success)
  console.log('Transaction completion status: ', await actor.call(txnActor, 'txnComplete'), '\n')
  if (verbose) { console.log("New order txn complete") }
}

async function paymentTxn(wId, dId, cId) {
  const amount = await getRandomInt(1, 5000)
  const txnActor = actor.proxy('PaymentTxn', uuidv4())
  var txn = {}
  txn.wId = wId, txn.dId = dId, txn.cId = cId
  txn.amount = amount
  const success = await actor.call(txnActor, 'startTxn', txn)
  console.log('Transaction success status: ', success)
  console.log('Transaction completion status: ', await actor.call(txnActor, 'txnComplete'), '\n')
  if (verbose) { console.log("Payment complete") }
  return amount
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

async function deliveryTxn(wId, carrierId, deliveryDate) {
  const txnActor = actor.proxy('DeliveryTxn', uuidv4())
  var txn = {}
  txn.wId = wId, txn.carrierId = carrierId, txn.deliveryDate = deliveryDate
  const success = await actor.call(txnActor, 'startTxn', txn)
  console.log('Transaction success status: ', success)
  console.log('Transaction completion status: ', await actor.call(txnActor, 'txnComplete'), '\n')
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

async function newOrderConsistencyCheck() {
  const district = actor.proxy('District', wId + ':' + dId)
  const dNextOIdOld = await actor.call(district, 'get', 'nextOId')

  const customer = actor.proxy('Customer', wId + ':' + dId + ':' + cId)
  const cLastOIdOld = wId + ':' + dId + ':' + 'o' + (dNextOIdOld.val + 1)

  const itemIds = ['i1', 'i2'], quantity = 10
  const itemKeys = ['quantity', 'ytd', 'orderCnt']
  let oldItemDetails = {}
  for (let i in itemIds) {
    const itemActor = actor.proxy('ItemStock', itemIds[i] + ':' + wId)
    oldItemDetails[itemIds[i]] = await actor.call(itemActor, 'getMultiple', itemKeys)
  }

  await newOrderTxn(wId, dId, cId, itemIds, quantity)

  const dNextOIdNew = await actor.call(district, 'get', 'nextOId')
  console.assert(dNextOIdNew.val == dNextOIdOld.val + 1, 
                "Next order id must increase by exactly 1 count.")

  const cLastOIdNew = await actor.call(customer, 'get', 'lastOId')
  console.assert(cLastOIdNew.val == cLastOIdOld,
                "Customer's last ordered id must be same as the lastest order id.")
  for (let i in itemIds) {
    const itemActor = actor.proxy('ItemStock', itemIds[i]+ ':' +  wId)
    const newDetails = await actor.call(itemActor, 'getMultiple', itemKeys)
    console.assert(newDetails.ytd.val == oldItemDetails[itemIds[i]].ytd.val + quantity, 
                  "Item ytd must increase exactly by new order's quantity.")
    console.assert(newDetails.orderCnt.val == oldItemDetails[itemIds[i]].orderCnt.val + 1, 
                  "Item's order count must increase exactly by one.")
  }
}

async function paymentConsistencyCheck() {
  const warehouse = actor.proxy('Warehouse', wId)
  const district = actor.proxy('District', wId + ':' + dId)
  const customer = actor.proxy('Customer', wId + ':' + dId + ':' + cId)

  const wYTDOld = await actor.call(warehouse, 'get', 'ytd')
  const dYTDOld = await actor.call(district, 'get', 'ytd')
  const cDetailsOld = await actor.call(customer, 'getMultiple', ['balance', 'ytdPayment', 'paymentCnt'])

  const amt = await paymentTxn(wId, dId, cId)
  // await new Promise(resolve => setTimeout(resolve, 2000));
  const wYTDNew = await actor.call(warehouse, 'get', 'ytd')
  const dYTDNew = await actor.call(district, 'get', 'ytd')
  const cDetailsNew = await actor.call(customer, 'getMultiple', ['balance', 'ytdPayment', 'paymentCnt'])

  console.assert(wYTDNew.val == wYTDOld.val + amt, 
                "Warehouse YTD after a payment txn must increase by payment amount.")
  console.assert(dYTDNew.val == dYTDOld.val + amt, 
                "District YTD after a payment txn must increase by payment amount.")
  console.assert(cDetailsNew.balance.val == cDetailsOld.balance.val - amt, 
                "Customer balance does not reflect payment.")
  console.assert(cDetailsNew.ytdPayment.val == cDetailsOld.ytdPayment.val + amt, 
                "Customer ytd payment does not reflect payment.")
  console.assert(cDetailsNew.paymentCnt.val == cDetailsOld.paymentCnt.val + 1, 
                "Customer payment count did not increase by 1.")
}

async function getTotalOrderAmount(oDetails) {
  let totalAmt = 0
  for (let i in oDetails.orderLines.val) {
    totalAmt += oDetails.orderLines.val[i].amount
  }
  return totalAmt
}

async function deliveryConsistencyCheck() {
  var dId, district, dDetailsOld
  for (let i = 1; i <= c.NUM_DISTRICTS; i++) {
    dId = 'd' + i
    district = actor.proxy('District', wId + ':' + dId)
    dDetailsOld = await actor.call(district, 'getMultiple', ['nextOId', 'lastDlvrOrd'])
    if (dDetailsOld.nextOId.val == 1 || 
      dDetailsOld.lastDlvrOrd.val == dDetailsOld.nextOId.val - 1) {
      // This implies either no order was placed in this district
      // or all orders in the district are delivered; skip district
      continue
    } else { break}
  }
  if (dId == null) { return }
  const orderId = wId + ':' + dId + ':'+ 'o' + Number(dDetailsOld.lastDlvrOrd.val+1)
  const order = actor.proxy('Order', orderId)
  const oCId = await actor.call(order, 'get', 'cId')

  const customer = actor.proxy('Customer', wId + ':' + dId + ':' + oCId)
  const cDetailsOld = await actor.call(customer, 'getMultiple', ['balance', 'deliveryCnt'])  
  const carrierId = 5, date = new Date()
  await deliveryTxn(wId, carrierId, date)

  const dlastDlvrOrdNew = await actor.call(district, 'get', 'lastDlvrOrd')
  console.assert(dlastDlvrOrdNew.val == dDetailsOld.lastDlvrOrd.val + 1, 
                "Last delivered order id must increase by exactly 1 count.")

  const oDetails = await actor.call(order, 'getMultiple', ['carrierId', 'orderLines'])
  console.assert(oDetails.carrierId.val == carrierId, 
              "Order carrier id must reflect the id sent in the transaction.")
  for( let i in oDetails.orderLines.val) {
    const ol = oDetails.orderLines.val[i]
    console.assert(ol.deliveryDate == JSON.stringify(date).replace(/['"]+/g, ''), 
                  "Order line's delivery date must reflect the date sent in the transaction.")
  }
  const amt = await getTotalOrderAmount(oDetails)
  const cDetailsNew = await actor.call(customer, 'getMultiple', ['balance', 'deliveryCnt'])
  console.assert(cDetailsNew.balance.val == cDetailsOld.balance.val + amt, 
    "Customer balance does not reflect delivery amount.")
  console.assert(cDetailsNew.deliveryCnt.val == cDetailsOld.deliveryCnt.val + 1, 
      "Customer delivery count did not increase by 1.")
}


async function main () {
  await newOrderConsistencyCheck()
  await paymentConsistencyCheck()
  await deliveryConsistencyCheck()
  console.log('Terminating sidecar')
  await sys.shutdown()
}

main()
