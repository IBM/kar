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

class PaymentTxn extends t.Transaction {
  async activate () {
    await super.activate()
  }

  async prepareWarehouse(wId, txnId) {
    const warehouse = actor.proxy('Warehouse', wId)
    const wKeys = { rw:['ytd'] }
    return [warehouse, await actor.call(warehouse, 'prepare', txnId, wKeys)]
  }

  async getDistrictDetails(wId, dId) {
    const district = actor.proxy('District', wId + ':' + dId)
    return [district, await actor.call(district, 'getMultiple', ['ytd'])]
  }

  async getCustomerDetails(wId, dId, cId) {
    const customer = actor.proxy('Customer', wId + ':' + dId + ':' + cId)
    const keys = ['balance', 'ytdPayment', 'paymentCnt']
    return [customer, await actor.call(customer, 'getMultiple', keys)]
  }

  async updateCustomerDetails(cDetails, amount) {
    let updatedCDetails = {}
    // Update customer details based on txn payment.
    updatedCDetails.balance = cDetails.balance - amount
    updatedCDetails.ytdPayment = cDetails.ytdPayment + amount
    updatedCDetails.paymentCnt = cDetails.paymentCnt + 1
    return updatedCDetails
  }

  async prepareTxn(txn) {
    let actorUpdates = {}, decision
    /* actorUpdates = {
      'warehouse': { 'actr': actorInst, 'update': { key: val },
      'district': { 'actr': actorInst, 'update': { key: val },
      'customer': { 'actr': actorInst, 'update': { key: val }
    } */
    const warehouse = actor.proxy('Warehouse', txn.wId)
    actorUpdates.warehouse = { actr: warehouse }

    const district = actor.proxy('District', txn.wId + ':' + txn.dId)
    actorUpdates.district = { actr: district }

    const customer = actor.proxy('Customer', txn.wId + ':' + txn.dId + ':' + txn.cId)
    actorUpdates.customer = { actr: customer }

    const prepared = await super.prepareTxn(actorUpdates, 'preparePayment')
    decision = prepared.decision, actorUpdates = prepared.actorUpdates

    if (decision) {
      const wDetails = actorUpdates.warehouse.values
      actorUpdates.warehouse.update = { ytd : wDetails.ytd + txn.amount }
      const dDetails = actorUpdates.district.values
      actorUpdates.district.update =  { ytd: dDetails.ytd + txn.amount }
      actorUpdates.customer.update = await this.updateCustomerDetails(actorUpdates.customer.values, txn.amount)
    }
    await actor.state.setMultiple(this, {decision: decision, actorUpdates: actorUpdates} )
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

  async startTxnOld(txn) {
    let actors = [], operations = [] /* Track all actors and their respective updates;
                                      perform the updates in an atomic txn. */
    const wDetails = await this.getWarehouseDetails(txn.wId)
    const dDetails = await this.getDistrictDetails(txn.wId, txn.dId)
    const cDetails = await this.getCustomerDetails(txn.wId, txn.dId, txn.cId)

    let wUpdate = {ytd : wDetails[1].ytd}
    wUpdate.ytd.val += txn.amount
    actors.push(wDetails[0]), operations.push(wUpdate)

    let dUpdate = {ytd: dDetails[1].ytd}
    dUpdate.ytd.val += txn.amount
    actors.push(dDetails[0]), operations.push(dUpdate)

    const updatedCDetails = await this.updateCustomerDetails(cDetails[1], txn.amount)
    actors.push(cDetails[0]), operations.push(updatedCDetails)

    return await super.transact(actors, operations)
  }
}

exports.PaymentTxn = PaymentTxn