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
const verbose = process.env.VERBOSE

class NewOrder extends gp.GenericParticipant {
  async activate() {
    const that = await super.activate()
    this.noId = that.noId || this.kar.id
  }

  async deactivate () {
    if (verbose) { console.log('actor', this.noId, 'deactivate') }
    await actor.state.removeAll(this)
  }

  async prepare(txnId, order) {
    let localDecision = await super.prepare(txnId)
    if (localDecision != null) { /* This txn is already prepared. */ return localDecision }
    await super.writePrepared(txnId, true, {})
    return true // Always return true as the txn always adds a new order entry.
  }

  async commit(txnId, decision, order) {
    let continueCommit = await super.commit(txnId, decision)
    if (!continueCommit) { /* This txn is already committed or not prepared. */ return }
    let writeMap = {}
    if (decision) { writeMap.noId = this.noId }
    await super.writeCommit(txnId, decision, writeMap)
    if (verbose) { console.log(`Committed transaction ${txnId}. Added new order ${this.noId}.\n`) }
    return
  }
}

class Order extends gp.GenericParticipant {
  async activate() {
    const that = await super.activate()
    this.oId = that.oId || this.kar.id
    this.cId = that.cId
    this.dId = that.dId
    this.wId = that.wId
    this.entryDate = that.entryDate || new Date()
    this.olCnt = that.olCnt || 0
    this.orderLines = that.orderLines || await super.createVal({})
    this.allLocal = true
    this.carrierId = that.carrierId || await super.createVal(null)
  }

  async getOrder() {
    return await actor.state.getAll(this)
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
    if (verbose) { console.log(`${this.kar.id} committed transaction ${txnId}.\n`) }
    return
  }
}

// Server setup: register actors with KAR and start express
const app = express()
app.use(sys.actorRuntime({ Order, NewOrder }))
app.listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')