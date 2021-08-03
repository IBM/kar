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
var gp = require('../generic_participant.js')
var c = require('./constants.js')
const verbose = process.env.VERBOSE

class ItemStock extends gp.GenericParticipant {
  async activate() {
    const that = await super.activate()
    this.itemId = that.itemId || this.kar.id
    this.wId = that.wId
    this.name = that.name || c.DEFAULT_ITEM_NAME
    this.price = that.price || c.DEFAULT_ITEM_PRICE
    this.quantity = that.quantity || { quantity:c.DEFAULT_QUANTITY, v:0 }
    this.ytd = that.ytd || { ytd:0, v:0 }
    this.orderCnt = that.orderCnt || { orderCnt:0, v:0 }
    this.remoteCnt = that.remoteCnt || { remoteCnt:0, v:0 }
    this.data = that.data
  }

  async addNewItem(item) {
    this.wId = item.wId
    this.name = item.name || c.DEFAULT_ITEM_NAME
    this.price = item.price || c.DEFAULT_ITEM_PRICE
    await actor.state.setMultiple(this, {wId : this.wId, name: this.name, price: this.price})
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
    console.log(`Committed transaction ${txnId}. Exact quantity left is ${this.quantity.quantity}.\n`)
    return
  }
}

// Server setup: register actors with KAR and start express
const app = express()
app.use(sys.actorRuntime({ ItemStock }))
app.listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')