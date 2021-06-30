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

const verbose = process.env.VERBOSE

class Transaction {
  async activate () {
    this.txnId = await actor.state.get(this, 'txnId') || uuidv4()
  }

  async transfer (sender, receiver, amt) {
    if (await actor.call(sender, 'updateBalance', this.txnId, amt)) {
      await actor.call(receiver, 'updateBalance', this.txnId, -amt)
      return true
    }
    return false
  }
}

class Account2 {
  async activate () {
    console.log("Here 2")
    this.currBalance = await actor.state.get(this, 'currBalance') || 1000
    this.txnIds = await actor.state.get(this, 'txnIds') || []
  }

  async getBalance() {
    return this.currBalance
  }

  async updateBalance (txnId, amt) {
    let txnIds = await actor.state.get(this, 'txnIds') || []
    if (txnIds.includes(txnId)) {
      // This account has already executed this transaction.
      return true
    }
    let success = false
    if ((this.currBalance - amt) >= 0) {
      this.currBalance -= amt
      this.txnIds.push(txnId)
      await actor.state.setMultiple(this, {currBalance: this.currBalance, txnIds: this.txnIds})
      console.log(`Updated ${this.kar.id}'s balance to ${this.currBalance}.`)
      success = true
    }
    return success
  }
}

// Server setup: register actors with KAR and start express
const app = express()
app.use(sys.actorRuntime({ Account2, Transaction }))
app.listen(process.argv[2], process.env.KAR_APP_HOST || '127.0.0.1')
