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
var t = require('../../txn_framework/transaction.js')
var c = require('../constants.js')
const verbose = process.env.VERBOSE

class DeliveryTxn extends t.Transaction {
  async activate () {
    await super.activate()
    this.actorUpdates = {}
  }

  async prepareDistricts(wId) {
    let districtActors = {}
    for (let i = 1; i <= c.NUM_DISTRICTS; i++) {
      let dId = 'd' + i
      districtActors[dId] = {}
      districtActors[dId].actr = actor.proxy('District', wId + ':' + dId)
    }
    const preparedActors = await super.prepareTxn(districtActors, 'prepareDelivery')
    this.actorUpdates = Object.assign({}, this.actorUpdates, preparedActors)
    let decision = true
    for (let i in this.actorUpdates) { decision = decision && this.actorUpdates[i].values.vote }
    return decision
  }

  async prepareOrder(wId) {
    let orderActors = {}
    for (let i in this.actorUpdates) {
      if (!i.startsWith('d')) { continue }
      const dDetails = this.actorUpdates[i].values
      if (dDetails.nextOId == 1 || dDetails.lastDlvrOrd >= dDetails.nextOId - 1) {
        // This implies either no order was placed in this district
        // or all orders in the district are delivered; skip district
        continue
      }
      const orderId = wId + ':' + i + ':'+ 'o' + Number(dDetails.lastDlvrOrd+1)
      orderActors[orderId] = {}
      orderActors[orderId].actr = actor.proxy('Order', orderId)
      actor.remove(actor.proxy('NewOrder', orderId))
    }
    if (Object.keys(orderActors).length == 0) { return false }
    const preparedActors = await super.prepareTxn(orderActors, 'prepareDelivery')
    this.actorUpdates = Object.assign({}, this.actorUpdates, preparedActors)
    let decision = true
    for (let i in this.actorUpdates) { decision = decision && this.actorUpdates[i].values.vote }
    return decision
  }

  async prepareCustomers(wId) {
    let custActors = {}
    for (let i in this.actorUpdates) { 
      if (!i.startsWith('w')) { continue }
      const cId = this.actorUpdates[i].values.cId, dId = this.actorUpdates[i].values.dId
      const custId = wId + ':' + dId + ':' + cId
      custActors[custId] = {}
      custActors[custId].actr = actor.proxy('Customer', custId)
    }
    const preparedActors = await super.prepareTxn(custActors, 'prepareDelivery')
    this.actorUpdates = Object.assign({}, this.actorUpdates, preparedActors)
  }

  async updateDistricts() {
    for (let i in this.actorUpdates) {
      if (!i.startsWith('d')) { continue }
      const dDetails = this.actorUpdates[i].values
      if (! (dDetails.nextOId == 1 || dDetails.lastDlvrOrd >= dDetails.nextOId - 1)) {
        this.actorUpdates[i].update =  { lastDlvrOrd: dDetails.lastDlvrOrd + 1}
      }
    }
  }

  async updateOrderDetails( carrierId, deliveryDate) {
    for (let i in this.actorUpdates) {
      if (!i.includes('o')) { continue }
      let updatedODetails = Object.assign({}, this.actorUpdates[i].values)
      updatedODetails.carrierId = carrierId
      for (let j in this.actorUpdates[i].values.orderLines) {
        updatedODetails.orderLines[j].deliveryDate = deliveryDate
      }
      this.actorUpdates[i].update = updatedODetails
    }
  }

  async updateCustomerDetails(wId) {
    for( let i in this.actorUpdates) {
      if (!i.includes('o')) { continue }
      let totalAmt = 0
      const oDetails = this.actorUpdates[i].values
      for (let j in oDetails.orderLines) {
        totalAmt += oDetails.orderLines[j].amount
      }
      const custId = wId + ':' + oDetails.dId + ':' + oDetails.cId
      this.actorUpdates[custId].update = {}
      this.actorUpdates[custId].update.deliveryCnt = this.actorUpdates[custId].values.deliveryCnt + 1
      this.actorUpdates[custId].update.balance = this.actorUpdates[custId].values.balance + totalAmt
    }
  }

  async prepareTxn(txn) {
    let earlyDecision = await this.prepareDistricts(txn.wId)
    if(!earlyDecision) { await actor.state.set(this, 'decision', earlyDecision); return earlyDecision }

    earlyDecision = await this.prepareOrder(txn.wId)
    if(!earlyDecision) { await actor.state.set(this, 'decision', earlyDecision); return earlyDecision }

    earlyDecision = await this.prepareCustomers(txn.wId)
    await Promise.all([this.updateDistricts(),
                      this.updateOrderDetails(txn.carrierId, txn.deliveryDate), 
                      this.updateCustomerDetails(txn.wId)])
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
    await actor.tell(this, 'sendCommitAsync', decision)
    return decision
  }
}

exports.DeliveryTxn = DeliveryTxn