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

const tp = require('../../txn_framework/txn_participant.js')
const c = require('../constants.js')

class District extends tp.TransactionParticipant {
  async activate () {
    const that = await super.activate()
    this.dId = that.dId || this.kar.id
    this.wId = that.wId || this.kar.id.split(':')[0]
    this.name = that.name || 'd-' + this.kar.id
    this.address = that.address || c.DEFAULT_ADDRESS
    this.tax = that.tax || c.DIST_TAX // Sales tax
    this.ytd = that.ytd || await super.createVal(0) // Year to date balance
    this.nextOId = that.nextOId || await super.createVal(1)
    this.lastDlvrOrd = that.lastDlvrOrd || await super.createVal(0)
  }

  async preparePayment (txnId) {
    // return {vote: false}
    let localDecision = await this.isTxnAlreadyPrepared(txnId)
    if (localDecision != null) { /* This txn is already prepared. */ return localDecision }
    localDecision = false
    if (this.ytd.rw === null && this.ytd.ro.length === 0) {
      this.ytd.rw = txnId
      localDecision = true
    }
    await super.writePrepared(txnId, localDecision, { ytd: this.ytd })
    return { vote: localDecision, ytd: this.ytd.val }
  }

  async prepareNewOrder (txnId) {
    const keys = ['nextOId', 'tax']
    return await this.prepare(txnId, keys)
  }

  async commitNewOrder (txnId, decision, update) {
    return await this.commit(txnId, decision, update)
  }

  async prepareDelivery (txnId) {
    let localDecision = await this.isTxnAlreadyPrepared(txnId)
    if (localDecision != null) { /* This txn is already prepared. */ return localDecision }
    localDecision = false
    if (this.nextOId.rw === null && this.nextOId.ro.length === 0 &&
      this.lastDlvrOrd.rw === null && this.lastDlvrOrd.ro.length === 0) {
      this.nextOId.ro.push(txnId)
      this.lastDlvrOrd.rw = txnId
      localDecision = true
    }
    await super.writePrepared(txnId, localDecision, { nextOId: this.nextOId, lastDlvrOrd: this.lastDlvrOrd })
    return { vote: localDecision, nextOId: this.nextOId.val, lastDlvrOrd: this.lastDlvrOrd.val }
  }
}

exports.District = District
