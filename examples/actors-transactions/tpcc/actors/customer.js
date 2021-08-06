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

const express = require('express')
const { actor, sys } = require('kar-sdk')
var gp = require('../../generic_participant.js')
var c = require('../constants.js')
var w = require('./warehouse.js')
Warehouse = w.Warehouse 
var d = require('./district.js')
District = d.District
const verbose = process.env.VERBOSE

class Customer extends gp.GenericParticipant {
  async activate() {
    const that = await super.activate()
    this.cId = that.cId || this.kar.id
    this.dId = that.dId
    this.wId = that.wId
    this.name = that.name || 'c-' + this.cId
    this.address = that.address || c.DEFAULT_ADDRESS
    this.credit = that.credit || 'GC' // 'GC' or 'BC' = good or bad credit
    this.creditLimit = that.creditLimit || 100
    this.discount = that.discount || 0
    this.balance = that.balance || { balance:c.DEFAULT_BALANCE, v:0 }
    this.ytdPayment = that.ytdPayment || { ytdPayment:0, v:0 } // Year to date payment
    this.paymentCnt = that.paymentCnt || { paymentCnt:0, v:0 }
    this.deliveryCnt = that.deliveryCnt || { deliveryCnt:0, v:0 }
    this.lastOId = that.lastOId || { lastOId: 0, v:0 }
  }

  async addCustomerToDistrict(dId, wId) {
    this.dId = dId, this.wId = wId
    await actor.state.setMultiple(this, {dId : this.dId, wId : this.wId})
  }

  async prepare(txnId, update) {
    let localDecision = await super.prepare(txnId)
    if (localDecision != null) { /* This txn is already prepared. */ return localDecision }
    localDecision = await super.checkVersionConflict(update)
    const writeMap = await super.createPrepareWriteMap(localDecision, update)
    await super.writePrepared(txnId, localDecision, writeMap)
    return localDecision
  }

  async commit(txnId, decision, update) {
    let continueCommit = await super.commit(txnId, decision)
    if (!continueCommit) { /* This txn is already committed or not prepared. */ return }
    const writeMap = await super.createCommitWriteMap(txnId, decision, update)
    await super.writeCommit(txnId, decision, writeMap)
    console.log(`${this.kar.id} committed transaction ${txnId}.\n`)
    return
  }
}

// Server setup: register actors with KAR and start express
const app = express()
app.use(sys.actorRuntime({ Warehouse, District, Customer }))
app.listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')