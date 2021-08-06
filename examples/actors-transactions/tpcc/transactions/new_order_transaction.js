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
var t = require('../../transaction.js')
var c = require('../constants.js')
const verbose = process.env.VERBOSE

class NewOrderTxn extends t.Transaction {
  async activate () {
    const that = await super.activate()
  }

  async getWarehouseDetails(wId) {
    const warehouse = actor.proxy('Warehouse', wId)
    return [warehouse, await actor.call(warehouse, 'getMultiple', ['tax'])]
  }

  async getDistrictDetails(wId, dId) {
    const district = actor.proxy('District', wId + ':' + dId)
    return [district, await actor.call(district, 'getMultiple', ['tax', 'nextOId'])]
  }

  async getCustomerDetails(wId, dId, cId) {
    const customer = actor.proxy('Customer', wId + ':' + dId + ':' + cId)
    return [customer, await actor.call(customer, 'getMultiple', ['discount', 'credit', 'lastOId'])]
  }

  async getItemDetails(itemId, supplyWId) {
    const itemStock = actor.proxy('ItemStock', itemId + ':' + supplyWId)
    const keys = ['price', 'name', 'quantity', 'ytd', 'orderCnt', 'version']
    return [itemStock, await actor.call(itemStock, 'getMultiple', keys)]
  }

  async getItemDetailsToWrite(itemDetails, ol) {
    let itemDetailsToWrite = Object.assign({}, itemDetails)
    // Update item details based on order
    const updatedQuantity = (itemDetails.quantity.quantity - ol.quantity) > 0? 
          (itemDetails.quantity.quantity - ol.quantity) : (itemDetails.quantity.quantity - ol.quantity + 91)
    itemDetailsToWrite.quantity.quantity = updatedQuantity
    itemDetailsToWrite.ytd.ytd = itemDetails.ytd.ytd + ol.quantity
    itemDetailsToWrite.orderCnt.orderCnt = itemDetails.orderCnt.orderCnt + 1
    return itemDetailsToWrite
  }

  async startTxn(txn) {
    let actors = [], operations = [] /* Track all actors and their respective updates;
                                      perform the updates in an atomic txn. */
    const wDetails = await this.getWarehouseDetails(txn.wId)
    const dDetails = await this.getDistrictDetails(txn.wId, txn.dId)
    const cDetails = await this.getCustomerDetails(txn.wId, txn.dId, txn.cId)
    
    const orderId = txn.wId + ':' + txn.dId + ':' + 'o' + dDetails[1].nextOId.nextOId
    const dUpdate = {nextOId: {nextOId: dDetails[1].nextOId.nextOId+1, v: dDetails[1].nextOId.v}}
    actors.push(dDetails[0]), operations.push(dUpdate)

    const cUpdate = {lastOId: {lastOId: orderId, v: cDetails[1].lastOId.v}}
    actors.push(cDetails[0]), operations.push(cUpdate)

    const order = actor.proxy('Order', orderId) // Create an order
    actors.push(order), operations.push(txn)

    const newOrder = actor.proxy('NewOrder', orderId) // Create a new order entry
    actors.push(newOrder), operations.push({})

    let totalAmount = 0
    for (let i in txn.orderLines.orderLines) {
      let ol = txn.orderLines.orderLines[i]
      const itemDetails = await this.getItemDetails(ol.itemId, ol.supplyWId)
      const itemDetailsToWrite = await this.getItemDetailsToWrite(itemDetails[1], ol)
      actors.push(itemDetails[0]), operations.push(itemDetailsToWrite)
      ol.amount = ol.quantity * itemDetails[1].price
      totalAmount += ol.amount
    }
    totalAmount = totalAmount * (1 - cDetails[1].discount) * (1 + wDetails[1].wTax + dDetails[1].tax)
    await super.transact(actors, operations)
  }
}

exports.NewOrderTxn = NewOrderTxn