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

const { actor } = require('kar-sdk')
const tp = require('../../txn_framework/txn_participant.js')
const c = require('../constants.js')

class Customer extends tp.TransactionParticipant {
  async activate () {
    const that = await super.activate()
    this.cId = that.cId || this.kar.id
    this.dId = that.dId || this.kar.id.split(':')[1]
    this.wId = that.wId || this.kar.id.split(':')[0]
    this.name = that.name || 'c-' + this.cId
    this.address = that.address || c.DEFAULT_ADDRESS
    this.credit = that.credit || 'GC' // 'GC' or 'BC' = good or bad credit
    this.creditLimit = that.creditLimit || 100
    this.discount = that.discount || 0
    this.balance = that.balance || await super.createVal(c.DEFAULT_BALANCE)
    this.ytdPayment = that.ytdPayment || await super.createVal(0) // Year to date payment
    this.paymentCnt = that.paymentCnt || await super.createVal(0)
    this.deliveryCnt = that.deliveryCnt || await super.createVal(0)
    this.lastOId = that.lastOId || await super.createVal(0)
  }

  async addCustomerToDistrict (dId, wId) {
    this.dId = dId
    this.wId = wId
    await actor.state.setMultiple(this, { dId: this.dId, wId: this.wId })
  }

  async preparePayment (txnId) {
    const keys = ['balance', 'ytdPayment', 'paymentCnt']
    return await this.prepare(txnId, keys)
  }

  async prepareNewOrder (txnId) {
    const keys = ['discount', 'credit', 'lastOId']
    return await this.prepare(txnId, keys)
  }

  async commitNewOrder (txnId, decision, update) {
    return await this.commit(txnId, decision, update)
  }

  async prepareDelivery (txnId) {
    const keys = ['balance', 'deliveryCnt']
    return await this.prepare(txnId, keys)
  }

  async prepareOrderStatus (txnId) {
    const keys = ['balance', 'name', 'lastOId']
    let localDecision = await super.isTxnAlreadyPrepared(txnId)
    if (localDecision != null) { /* This txn is already prepared. */ return localDecision }
    localDecision = this.checkForConflictRO(txnId, keys)
    const maps = this.createPrepareValueAndWriteMap(localDecision, keys)
    await this.writePrepared(txnId, localDecision, maps.writeMap)
    return maps.values
  }
}

exports.Customer = Customer
