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
var c = require('../constants.js')
const verbose = process.env.VERBOSE

class DeliveryTxn extends t.Transaction {
  async activate () {
    await super.activate()
    this.actorUpdates = {}
  }

  async getWarehouseDetails(wId) {
    const warehouse = actor.proxy('Warehouse', wId)
    return [warehouse, await actor.call(warehouse, 'getMultiple', ['ytd'])]
  }

  async prepareDistrict(wId, dId) {
    const district = actor.proxy('District', wId + ':' + dId)
    this.actorUpdates.district = { actr: district }
    await actor.state.set(this, 'actorUpdates', this.actorUpdates)
    return [district, await actor.call(district, 'prepareDelivery', this.txnId)]
  }

  async prepareCustomer(wId, dId, cId) {
    const customer = actor.proxy('Customer', wId + ':' + dId + ':' + cId)
    this.actorUpdates.customer = { actr: customer }
    await actor.state.set(this, 'actorUpdates', this.actorUpdates)
    return [customer, await actor.call(customer, 'prepareDelivery', this.txnId)]
  }

  async prepareOrder(oId) {
    const order = actor.proxy('Order', oId)
    this.actorUpdates.order = { actr: order }
    await actor.state.set(this, 'actorUpdates', this.actorUpdates)
    return [order, await actor.call(order, 'prepareDelivery', this.txnId)]
  }

  async getTotalOrderAmount(oDetails) {
    let totalAmt = 0
    for (let i in oDetails.orderLines) {
      totalAmt += oDetails.orderLines[i].amount
    }
    return totalAmt
  }

  async updateOrderDetails(oDetails, carrierId, deliveryDate) {
    let updatedODetails = Object.assign({}, oDetails)
    updatedODetails.carrierId = carrierId
    for (let i in oDetails.orderLines) {
      updatedODetails.orderLines[i].deliveryDate = deliveryDate
    }
    return updatedODetails
  }

  async updateCustomerDetails(cDetails, totalAmount) {
    let updatedCDetails = Object.assign({}, cDetails)
    updatedCDetails.balance += totalAmount
    updatedCDetails.deliveryCnt += 1
    return updatedCDetails
  }

  async prepareTxn(txn) {
    for (let i = 1; i <= c.NUM_DISTRICTS; i++) {
      let dId = 'd' + i
      const dDetails = await this.prepareDistrict(txn.wId, dId)
      if (dDetails[1].nextOId == 1 || 
        dDetails[1].lastDlvrOrd == dDetails[1].nextOId - 1) {
        // This implies either no order was placed in this district
        // or all orders in the district are delivered; skip district
        continue
      }
      this.actorUpdates.district = { actr: dDetails[0], values: dDetails[1] }
      this.actorUpdates.district.update =  { lastDlvrOrd: dDetails[1].lastDlvrOrd + 1}

      const orderId = txn.wId + ':' + dId + ':'+ 'o' + Number(dDetails[1].lastDlvrOrd+1)
      await actor.remove(actor.proxy('NewOrder', orderId))

      const oDetails = await this.prepareOrder(orderId)
      this.actorUpdates.order = { actr: oDetails[0], values: oDetails[1] }
      this.actorUpdates.order.update =  await this.updateOrderDetails(oDetails[1], txn.carrierId, txn.deliveryDate)
      const totalOrderAmt = await this.getTotalOrderAmount(oDetails[1])

      const cDetails = await this.prepareCustomer(txn.wId, dId, oDetails[1].cId)
      this.actorUpdates.customer = { actr: cDetails[0], values: cDetails[1] }
      this.actorUpdates.customer.update = await this.updateCustomerDetails(cDetails[1], totalOrderAmt)
    }
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
    for (let i = 1; i <= c.NUM_DISTRICTS; i++) {
      let dId = 'd' + i
      const dDetails = await this.getDistrictDetails(txn.wId, dId)
      if (dDetails[1].nextOId.val == 1 || 
        dDetails[1].lastDlvrOrd.val == dDetails[1].nextOId.val - 1) {
        // This implies either no order was placed in this district
        // or all orders in the district are delivered; skip district
        continue
      }
      const orderId = txn.wId + ':' + dId + ':'+ 'o' + Number(dDetails[1].lastDlvrOrd.val+1)
      await actor.remove(actor.proxy('NewOrder', orderId))

      let dUpdate = {lastDlvrOrd: dDetails[1].lastDlvrOrd}
      dUpdate.lastDlvrOrd.val += 1
      actors.push(dDetails[0]), operations.push(dUpdate)

      const oDetails = await this.getOrderDetails(orderId)
      const updatedODetails = await this.updateOrderDetails(oDetails[1], txn.carrierId, txn.deliveryDate)
      actors.push(oDetails[0]), operations.push(updatedODetails)
      const totalOrderAmt = await this.getTotalOrderAmount(oDetails[1])

      const cDetails = await this.getCustomerDetails(txn.wId, dId, oDetails[1].cId)
      const updatedCDetails = await this.updateCustomerDetails(cDetails[1], totalOrderAmt)
      actors.push(cDetails[0]), operations.push(updatedCDetails)
    }
    return await super.transact(actors, operations)
  }
}

exports.DeliveryTxn = DeliveryTxn