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
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

const tp = require('../../txn_framework/txn_participant.js')
const c = require('../constants.js')

class Warehouse extends tp.TransactionParticipant {
  async activate () {
    const that = await super.activate()
    this.wId = that.wId || this.kar.id
    this.name = that.name || 'w-' + this.kar.id
    this.address = that.address || c.DEFAULT_ADDRESS
    this.tax = that.tax || c.WAREHOUSE_TAX // Sales tax
    this.ytd = that.ytd || await super.createVal(0) // Year to date balance
  }

  async preparePayment (txnId) {
    const keys = ['ytd']
    return await this.prepare(txnId, keys)
  }

  async prepareNewOrder (txnId) {
    // Accessing only a read-only field, 'tax'
    return { vote: true, tax: this.tax }
  }

  async commitNewOrder (txnId, decision, update) {
    // No-op

  }
}

exports.Warehouse = Warehouse
