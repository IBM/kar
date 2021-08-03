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
const verbose = process.env.VERBOSE

class Warehouse extends gp.GenericParticipant {
  async activate () {
    const that = await super.activate()
    this.wId = that.wId || this.kar.id
    this.name = that.name || 'w-' + this.kar.id
    this.address = that.address || c.DEFAULT_ADDRESS
    this.tax = that.tax || c.WAREHOUSE_TAX // Sales tax
    this.ytd = that.ytd || { ytd:0, v:0 } // Year to date balance
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
    console.log(`Committed transaction ${txnId}. YTD is ${this.ytd.ytd}.\n`)
    return
  }
}

// Server setup: register actors with KAR and start express
// const app = express()
// app.use(sys.actorRuntime({ Warehouse }))
// app.listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')

exports.Warehouse = Warehouse