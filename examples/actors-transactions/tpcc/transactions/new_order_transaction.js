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
var t = require('../../transaction.js')
const verbose = process.env.VERBOSE

class NewOrderTxn extends t.Transaction {
  async activate () {
    await super.activate()
    this.actorUpdates = {}
  }

  async prepareWarehouse(wId) {
    const warehouse = actor.proxy('Warehouse', wId)
    return [warehouse, await actor.call(warehouse, 'prepareNewOrder', this.txnId)]
  }

  async prepareDistrict(wId, dId) {
    const district = actor.proxy('District', wId + ':' + dId)
    this.actorUpdates.district = { actr: district }
    await actor.state.set(this, 'actorUpdates', this.actorUpdates)
    return [district, await actor.call(district, 'prepareNewOrder', this.txnId)]
  }

  async prepareCustomer(wId, dId, cId) {
    const customer = actor.proxy('Customer', wId + ':' + dId + ':' + cId)
    this.actorUpdates.customer = { actr: customer }
    await actor.state.set(this, 'actorUpdates', this.actorUpdates)
    return [customer, await actor.call(customer, 'prepareNewOrder', this.txnId)]
  }

  async prepareItem(itemId, supplyWId) {
    const itemStock = actor.proxy('ItemStock', itemId + ':' + supplyWId)
    this.actorUpdates[itemId] = { actr: itemStock }
    await actor.state.set(this, 'actorUpdates', this.actorUpdates)
    return [itemStock, await actor.call(itemStock, 'prepareNewOrder', this.txnId)]
  }

  async getItemDetailsToWrite(itemDetails, ol) {
    let itemDetailsToWrite = Object.assign({}, itemDetails)
    // Update item details based on order
    const updatedQuantity = (itemDetails.quantity - ol.quantity) > 0? 
          (itemDetails.quantity - ol.quantity) : (itemDetails.quantity - ol.quantity + 91)
    itemDetailsToWrite.quantity = updatedQuantity
    itemDetailsToWrite.ytd = itemDetails.ytd + ol.quantity
    itemDetailsToWrite.orderCnt = itemDetails.orderCnt + 1
    return itemDetailsToWrite
  }

  async prepareTxn(txn) {
    const wDetails = await this.prepareWarehouse(txn.wId)
    this.actorUpdates.warehouse = { actr: wDetails[0], values: wDetails[1] }

    const dDetails = await this.prepareDistrict(txn.wId, txn.dId)
    this.actorUpdates.district = { actr: dDetails[0], values: dDetails[1] }
    this.actorUpdates.district.update =  { nextOId: dDetails[1].nextOId + 1}

    const orderId = txn.wId + ':' + txn.dId + ':' + 'o' + dDetails[1].nextOId
    const cDetails = await this.prepareCustomer(txn.wId, txn.dId, txn.cId)
    this.actorUpdates.customer = { actr: cDetails[0], values: cDetails[1] }
    this.actorUpdates.customer.update = { lastOId: orderId}
    
    let totalAmount = 0
    for (let i in txn.orderLines) {
      let ol = txn.orderLines[i]
      const itemDetails = await this.prepareItem(ol.itemId, ol.supplyWId)
      this.actorUpdates[ol.itemId] = {actr: itemDetails[0], values: itemDetails[1]}
      this.actorUpdates[ol.itemId].update = await this.getItemDetailsToWrite(itemDetails[1], ol)
      txn.orderLines[i].amount = ol.quantity * itemDetails[1].price
      totalAmount += ol.amount
    }
    totalAmount = totalAmount * (1 - cDetails[1].discount) * (1 + wDetails[1].wTax + dDetails[1].tax)
    
    const order = actor.proxy('Order', orderId) // Create an order
    const oDetails = await actor.call(order, 'prepareNewOrder', this.txnId)
    this.actorUpdates.order = {actr: order, values:oDetails, update: txn}

    const newOrder = actor.proxy('NewOrder', orderId) // Create a new order entry
    const noDetails = await actor.call(newOrder, 'prepareNewOrder', this.txnId)
    this.actorUpdates.newOrder = {actr: newOrder, values:noDetails}
  
    let decision = true
    for (let i in this.actorUpdates) { decision = decision && this.actorUpdates[i].values.vote }
    await actor.state.setMultiple(this, {decision: decision, actorUpdates: this.actorUpdates} )
    return decision
  }

  async startTxn(txn) {
    if (verbose) { console.log(`Begin transaction ${this.txnId}.`) }
    const that = await actor.state.getAll(this)
    if (that.commitComplete) { return that.decision }
    let decision = that.decision
    if (that.decision == null) {
      try {
        decision = await this.prepareTxn(txn)
      } catch (error) {
        console.log(error.toString())
        // If decision is not already set, abort this txn as something went wrong.
        if (await actor.state.get(this, 'decision') == null) { decision = false }
      }
    }
    await actor.tell(this, 'sendCommitAsync', decision, 'commitNewOrder')
    return decision
  }

  async startTxnOld(txn) {
    let actors = [], operations = [] /* Track all actors and their respective updates;
                                      perform the updates in an atomic txn. */
    const wDetails = await this.getWarehouseDetails(txn.wId)
    const dDetails = await this.getDistrictDetails(txn.wId, txn.dId)
    const cDetails = await this.getCustomerDetails(txn.wId, txn.dId, txn.cId)

    let dUpdate = {nextOId: dDetails[1].nextOId}
    dUpdate.nextOId.val += 1
    actors.push(dDetails[0]), operations.push(dUpdate)

    const orderId = txn.wId + ':' + txn.dId + ':' + 'o' + dDetails[1].nextOId.val
    let cUpdate = {lastOId: cDetails[1].lastOId} 
    cUpdate.lastOId.val = orderId
    actors.push(cDetails[0]), operations.push(cUpdate)
    
    const order = actor.proxy('Order', orderId) // Create an order
    actors.push(order), operations.push(txn)

    const newOrder = actor.proxy('NewOrder', orderId) // Create a new order entry
    actors.push(newOrder), operations.push({})

    let totalAmount = 0
    for (let i in txn.orderLines.val) {
      let ol = txn.orderLines.val[i]
      const itemDetails = await this.getItemDetails(ol.itemId, ol.supplyWId)
      const itemDetailsToWrite = await this.getItemDetailsToWrite(itemDetails[1], ol)
      actors.push(itemDetails[0]), operations.push(itemDetailsToWrite)
      ol.amount = ol.quantity * itemDetails[1].price
      totalAmount += ol.amount
    }
    totalAmount = totalAmount * (1 - cDetails[1].discount) * (1 + wDetails[1].wTax + dDetails[1].tax)
    return await super.transact(actors, operations)
  }
}

exports.NewOrderTxn = NewOrderTxn