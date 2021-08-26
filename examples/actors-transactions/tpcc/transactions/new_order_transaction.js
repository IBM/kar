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

  getItemDetailsToWrite(itemDetails, ol) {
    let itemDetailsToWrite = Object.assign({}, itemDetails)
    // Update item details based on order
    const updatedQuantity = (itemDetails.quantity - ol.quantity) > 0? 
          (itemDetails.quantity - ol.quantity) : (itemDetails.quantity - ol.quantity + 91)
    itemDetailsToWrite.quantity = updatedQuantity
    itemDetailsToWrite.ytd = itemDetails.ytd + ol.quantity
    itemDetailsToWrite.orderCnt = itemDetails.orderCnt + 1
    return itemDetailsToWrite
  }

  async prepareWDC(wId, dId, cId) {
    const warehouse = actor.proxy('Warehouse', wId)
    this.actorUpdates.warehouse = { actr: warehouse }

    const district = actor.proxy('District', wId + ':' + dId)
    this.actorUpdates.district = { actr: district }

    const customer = actor.proxy('Customer', wId + ':' + dId + ':' + cId)
    this.actorUpdates.customer = { actr: customer }

    this.actorUpdates = await super.prepareTxn(this.actorUpdates, 'prepareNewOrder')
    let decision = true
    for (let i in this.actorUpdates) { decision = decision && this.actorUpdates[i].values.vote }
    return decision
  }

  async prepareItemsAndOrders(orderLines, orderId) {
    let itemOrderActors = {}
    for (let i in orderLines) {
      let ol = orderLines[i]
      const itemActor = actor.proxy('ItemStock', ol.itemId + ':' + ol.supplyWId)
      itemOrderActors[ol.itemId] = {actr: itemActor}
    }
    const order = actor.proxy('Order', orderId)
    itemOrderActors.order = {actr: order}
    const newOrder = actor.proxy('NewOrder', orderId) // Create a new order entry
    itemOrderActors.newOrder = {actr: newOrder}

    const preparedActors = await super.prepareTxn(itemOrderActors, 'prepareNewOrder')
    this.actorUpdates = Object.assign({}, this.actorUpdates, preparedActors)
  }

  async prepareTxn(txn) {
    let earlyDecision = await this.prepareWDC(txn.wId, txn.dId, txn.cId)
    if(!earlyDecision) { await actor.state.set(this, 'decision', earlyDecision); return earlyDecision }
    const wDetails = this.actorUpdates.warehouse.values
    const dDetails = this.actorUpdates.district.values
    const cDetails = this.actorUpdates.customer.values

    this.actorUpdates.district.update =  { nextOId: dDetails.nextOId + 1}

    const orderId = txn.wId + ':' + txn.dId + ':' + 'o' + dDetails.nextOId
    this.actorUpdates.customer.update = { lastOId: orderId }
  
    await this.prepareItemsAndOrders(txn.orderLines, orderId)
    let totalAmount = 0
    for (let i in txn.orderLines) {
      let ol = txn.orderLines[i]
      const itemDetails = this.actorUpdates[ol.itemId].values
      this.actorUpdates[ol.itemId].update = this.getItemDetailsToWrite(itemDetails, ol)
      txn.orderLines[i].amount = ol.quantity * itemDetails.price
      totalAmount += ol.amount
    }
    totalAmount = totalAmount * (1 - cDetails.discount) * (1 + wDetails.wTax + dDetails.tax)
    this.actorUpdates.order.update = txn
  
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
}

exports.NewOrderTxn = NewOrderTxn