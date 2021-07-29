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
const { v4: uuidv4 } = require('uuid')
var gp = require('../generic_participant.js')
var c = require('./constants.js')
const verbose = process.env.VERBOSE

class District extends gp.GenericParticipant {
  async activate () {
    const that = await super.activate()
    this.dId = that.dId || this.kar.id
    console.log(this.dId)
    this.wId = that.wId
    this.name = that.name || 'd-' + this.kar.id
    this.address = that.address || c.DEFAULT_ADDRESS
    this.tax = that.tax || c.DIST_TAX // Sales tax
    this.ytd = that.ytd || 0 // Year to date balance
    this.nextOId = that.nextOId || 0
    this.reservedNextOId = that.reservedNextOId || 0
  }

  async prepare(txnId, currentOId) {
    let localDecision = await super.prepare(txnId)
    if (localDecision != null) { /* This txn is already prepared. */ return localDecision }
    // The read order id, currentOId, must be strictly match the actor's reserved order id
    // in case any other txn is concurrently preparing this actor. 
    (this.reservedNextOId == currentOId) ? localDecision = true: localDecision = false
    if (localDecision) { this.reservedNextOId +=1 }
    await super.writePrepared(txnId, localDecision, {reservedNextOId: this.reservedNextOId})
    return localDecision
  }

  async commit(txnId, decision, currentOId) {
    let continueCommit = await super.commit(txnId, decision)
    if (!continueCommit) { /* This txn is already committed or not prepared. */ return }
    if (decision) {  this.nextOId +=1  }
    if (!decision && await super.getTxnLocalDecision(txnId)) { this.reservedNextOId -=1 }
    await super.writeCommit(txnId, decision, {nextOId: this.nextOId,
                            reservedNextOId: this.reservedNextOId} )
    console.log(`Committed transaction ${txnId}. Next order id is ${this.nextOId}.\n`)
    return
  }
}

// Server setup: register actors with KAR and start express
// const app = express()
// app.use(sys.actorRuntime({ District }))
// app.listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')

exports.District = District