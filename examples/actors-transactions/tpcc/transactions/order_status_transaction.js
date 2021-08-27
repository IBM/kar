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
const verbose = process.env.VERBOSE

class OrderStatusTxn extends t.Transaction {
  async activate () {
    await super.activate()
    this.actorUpdates = {}
  }

  async prepareCustomer(wId, dId, cId) {
    const customer = actor.proxy('Customer', wId + ':' + dId + ':' + cId)
    this.actorUpdates.customer = { actr: customer }
    await actor.state.set(this, 'actorUpdates', this.actorUpdates)
    return [customer, await actor.call(customer, 'prepareOrderStatus', this.txnId)]
  }

  async prepareOrder(oId) {
    const order = actor.proxy('Order', oId)
    this.actorUpdates.order = { actr: order }
    await actor.state.set(this, 'actorUpdates', this.actorUpdates)
    return [order, await actor.call(order, 'prepareOrderStatus', this.txnId)]
  }

  async prepareTxn(txn) {
    const cDetails = await this.prepareCustomer(txn.wId, txn.dId, txn.cId)
    this.actorUpdates.customer = { actr: cDetails[0], values: cDetails[1] }
    const oDetails = await this.prepareOrder(cDetails[1].lastOId)
    this.actorUpdates.order = { actr: oDetails[0], values: oDetails[1] }
    
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
    const actorUpdates = await actor.state.get(this, 'actorUpdates')
    return { decision, orderDetails: actorUpdates.order.values }
  }
}

exports.OrderStatusTxn = OrderStatusTxn