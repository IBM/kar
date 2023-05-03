/*
 * Copyright IBM Corporation 2020,2023
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

const { actor } = require('kar-sdk')
const tp = require('../../txn_framework/txn_participant.js')
const verbose = process.env.VERBOSE

class NewOrder extends tp.TransactionParticipant {
  async activate () {
    const that = await super.activate()
    this.noId = that.noId || this.kar.id
  }

  async deactivate () {
    if (verbose) { console.log('actor', this.noId, 'deactivate') }
    await actor.state.removeAll(this)
  }

  async prepareNewOrder (txnId) {
    const localDecision = await this.isTxnAlreadyPrepared(txnId)
    if (localDecision != null) { /* This txn is already prepared. */ return localDecision }
    await super.writePrepared(txnId, true, {})
    return { vote: true } // Always return true as the txn always adds a new order entry.
  }

  async commitNewOrder (txnId, decision, update) {
    return await this.commit(txnId, decision, update)
  }

  async prepare (txnId) {
    const localDecision = await this.isTxnAlreadyPrepared(txnId)
    if (localDecision != null) { /* This txn is already prepared. */ return localDecision }
    await super.writePrepared(txnId, true, {})
    return { vote: true } // Always return true as the txn always adds a new order entry.
  }

  async commit (txnId, decision, order) {
    const continueCommit = await this.isTxnAlreadyCommitted(txnId)
    if (!continueCommit) { /* This txn is already committed or not prepared. */ return }
    const writeMap = {}
    if (decision) { writeMap.noId = this.noId }
    await this.writeCommit(txnId, decision, writeMap)
    if (verbose) { console.log(`Committed transaction ${txnId}. Added new order ${this.noId}.\n`) }
  }
}

class Order extends tp.TransactionParticipant {
  async activate () {
    const that = await super.activate()
    this.oId = that.oId || this.kar.id
    this.cId = that.cId
    this.dId = that.dId || this.kar.id.split(':')[1]
    this.wId = that.wId || this.kar.id.split(':')[0]
    this.entryDate = that.entryDate || new Date()
    this.olCnt = that.olCnt || 0
    this.orderLines = that.orderLines || await super.createVal({})
    this.allLocal = true
    this.carrierId = that.carrierId || await super.createVal(0)
  }

  async getOrder () {
    return await actor.state.getAll(this)
  }

  async prepareNewOrder (txnId) {
    const localDecision = await this.isTxnAlreadyPrepared(txnId)
    if (localDecision != null) { /* This txn is already prepared. */ return localDecision }
    await super.writePrepared(txnId, true, {})
    return { vote: true } // Always return true as the txn always adds a new order entry.
  }

  async commitNewOrder (txnId, decision, update) {
    return await this.commit(txnId, decision, update)
  }

  async prepareDelivery (txnId) {
    const keys = ['cId', 'dId', 'orderLines', 'carrierId']
    let localDecision = false
    try {
      localDecision = await this.prepare(txnId, keys)
    } catch {
      await this.writePrepared(txnId, localDecision, {})
    }
    return localDecision
  }

  async prepareOrderStatus (txnId) {
    const keys = ['entryDate', 'carrierId', 'orderLines']
    let localDecision = await super.isTxnAlreadyPrepared(txnId)
    if (localDecision != null) { /* This txn is already prepared. */ return localDecision }
    localDecision = this.checkForConflictRO(txnId, keys)
    const maps = this.createPrepareValueAndWriteMap(localDecision, keys)
    await this.writePrepared(txnId, localDecision, maps.writeMap)
    return maps.values
  }
}

exports.NewOrder = NewOrder
exports.Order = Order
