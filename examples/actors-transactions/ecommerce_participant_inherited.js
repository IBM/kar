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
var gp = require('./generic_participant.js')
const verbose = process.env.VERBOSE

class Item extends gp.GenericParticipant {
  async activate () {
    const that = await super.activate()
    this.quantity = that.quantity || 1000
    this.reservedQuantity = that.reservedQuantity || 0
  }

  async setQuantity(quantity) {
    this.quantity = quantity
    actor.state.set(this, 'quantity', this.quantity)
  }

  async getAvailableQuantity() {
    return this.quantity - this.reservedQuantity
  }

  async getExactQuantity() {
    return this.quantity
  }

  async prepare(txnId, order) {
    let prepared = await super.prepare(txnId)
    if (prepared != null) { /* This txn is already prepared. */ return prepared }
    prepared = ((this.quantity - (this.reservedQuantity + order)) > 0)
    if (prepared) { this.reservedQuantity += order }
    console.log('Reserverd quantity: ', this.reservedQuantity)
    await super.writePrepared(txnId, prepared, {reservedQuantity: this.reservedQuantity})
    return prepared
  }
  
  async commit(txnId, decision, order) {
    let continueCommit = await super.commit(txnId, decision)
    if (!continueCommit) { /* This txn is already committed or not prepared. */ return }
    if (decision == true) { this.quantity -= order }
    if (await super.isTxnPreparedTrue(txnId)) { this.reservedQuantity -= order }
    await super.writeCommit( {quantity: this.quantity, reservedQuantity: this.reservedQuantity} )
    console.log(`Committed transaction ${txnId}. Exact quantity left is ${this.quantity}.\n`)
    return
  }
}

// Server setup: register actors with KAR and start express
const app = express()
app.use(sys.actorRuntime({ Item1:Item, Item2:Item, Item3:Item }))
app.listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')