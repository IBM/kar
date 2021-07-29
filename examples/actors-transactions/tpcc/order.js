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
const verbose = process.env.VERBOSE

class OrderLine {
  constructor (olNum, itemId, quantity, supplyWId, amount) {
    this.olNumber = olNum
    this.itemId = itemId
    this.quantity = quantity
    this.supplyWId = supplyWId
    this.amount = amount
    
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
    this.orderLines = that.orderLines || {}
    this.allLocal = true
  }

  async prepare(txnId, order) {
    let localDecision = await super.prepare(txnId)
    if (localDecision != null) { /* This txn is already prepared. */ return localDecision }
    this.cId = order.cId, this.dId = order.dId, this.wId = order.wId
    this.olCnt = order.olCnt
    for (let key in order.orderLines) {
      const ol = order.orderLines[key]
      let orderLine = new OrderLine(key, ol.itemId, ol.quantity, ol.supplyWId, ol.amount)
      this.orderLines[key] = orderLine
    }
    await super.writePrepared(txnId, true, {cId: this.cId, dId : this.dId, wId : this.wId,
          olCnt: this.olCnt, orderLines: this.orderLines})
    return true
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
const app = express()
app.use(sys.actorRuntime({ Order }))
app.listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')