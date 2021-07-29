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

const express = require('express')
const { actor, sys } = require('kar-sdk')
const { v4: uuidv4 } = require('uuid')
var t = require('../transaction.js')
var c = require('./constants.js')
const verbose = process.env.VERBOSE

class NewOrderTxn extends t.Transaction {
  async activate () {
    const that = await super.activate()
  }

  async getItemDetails(itemStock) {
    const keys = ['price', 'name', 'quantity', 'ytd', 'orderCnt']
    return await actor.call(itemStock, 'getMultiple', keys)
  }

  async startTxn(txn) {
    let actors = [], operations = [] /* Track all actors and their respective updates;
                                      perform the updates in an atomic txn. */
    const warehouse = actor.proxy('Warehouse', txn.wId)
    const wTax = await actor.call(warehouse, 'get', 'tax')

    const district = actor.proxy('District', txn.dId + ':' + txn.wId)
    const distDetails = await actor.call(district, 'getMultiple', ['tax', 'nextOId'])
    actors.push(district), operations.push(distDetails.nextOId)

    const customer = actor.proxy('Customer', txn.cId + ':' + txn.dId + ':' + txn.wId)
    const custDetails = await actor.call(customer, 'getMultiple', ['discount', 'credit'])

    const order = actor.proxy('Order', distDetails.nextOId+1)
    actors.push(order), operations.push(txn)

    let totalAmount = 0
    for (let i in txn.orderLines) {
      let ol = txn.orderLines[i]
      const itemStock = actor.proxy('ItemStock', ol.itemId + ':' + ol.supplyWId)
      const itemDetails = await this.getItemDetails(itemStock)
      let itemDetailsToWrite = Object.assign({}, itemDetails)
      // Update item details based on order
      const updatedQuantity = (itemDetails.quantity - ol.quantity) > 0? 
            (itemDetails.quantity - ol.quantity) : (itemDetails.quantity - ol.quantity + 91)
      itemDetailsToWrite.quantity = updatedQuantity
      itemDetailsToWrite.ytd = itemDetails.ytd + ol.quantity
      itemDetailsToWrite.orderCnt = itemDetails.orderCnt + 1
      ol.amount = ol.quantity * itemDetails.price
      totalAmount += ol.amount
      actors.push(itemStock), operations.push(itemDetailsToWrite)
    }
    totalAmount = totalAmount * (1 - custDetails.discount) * (1 + wTax + distDetails.tax)
    await super.transact(actors, operations)
  }
}

async function main() {
  const txnActor = actor.proxy('NewOrderTxn', uuidv4())
  var txn = {}
  txn.wId = 'w1', txn.dId = 'd2', txn.cId = 'c1'
  txn.olCnt = 1
  txn.orderLines = {
    '1': {itemId: 'i2', supplyWId: 'w1', quantity: 10}
  }
  await actor.call(txnActor, 'startTxn', txn)
}

// Server setup: register actors with KAR and start express
const app = express()
app.use(sys.actorRuntime({ NewOrderTxn }))
app.listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')

main()