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

class Participant {
  async activate () {
    const that = await actor.state.getAll(this)
    this.exactBalance = that.exactBalance || 1000
    this.credits = that.credits || 0
    this.debits =  that.debits || 0
    this.preparedTxns = that.preparedTxns || {}
    this.committedTxns = that.committedTxns || {}
  }

  async setBalance(balance) {
    this.exactBalance = balance
    actor.state.set(this, 'exactBalance', this.exactBalance)
  }

  async getAvailableBalance() {
    return this.exactBalance - this.debits
  }

  async getExactBalance() {
    return this.exactBalance
  }

  async prepare(txnId, amt) {
    console.log(`Received prepare for txn ${txnId}`)
    this.preparedTxns = await actor.state.get(this, 'preparedTxns') || {}
    if (txnId in this.preparedTxns) {
      // Already prepared this txn.
      return this.preparedTxns[txnId]
    }
    const prepared = (this.exactBalance + amt > 0)
    this.preparedTxns[txnId] = prepared
    if (prepared) { amt < 0? this.debits -= amt : this.credits += amt }
    console.log('Debits: ', this.debits, " Credits: ", this.credits)
    await actor.state.setMultiple(this, {debits: this.debits, credits: this.credits,
      preparedTxns: this.preparedTxns})
    return prepared
  }
  
  async commit(txnId, decision, amt) {
    // await new Promise(r => setTimeout(r, 5000));
    if (!(txnId in this.preparedTxns)) { throw new Error('An unprepared txn cannot be committed') }
    console.log(`Received commit for txn ${txnId}`)
    this.committedTxns = await actor.state.get(this, 'committedTxns') || {}
    if (txnId in this.committedTxns) {
      // Already committed this txn.
      return
    } 
    console.log(`Transaction ${txnId}'s decision is`, decision)
    this.committedTxns[txnId] = decision
    if (decision == true) {
      this.exactBalance += amt
    }
    if (this.preparedTxns[txnId]) { amt < 0? this.debits += amt : this.credits -= amt }
    await actor.state.setMultiple(this, {exactBalance: this.exactBalance, debits: this.debits,
      credits: this.credits, committedTxns : this.committedTxns})
    console.log(`Committed transaction ${txnId}. Exact balance is ${this.exactBalance}.\n`)
    return
  }
}

// Server setup: register actors with KAR and start express
const app = express()
app.use(sys.actorRuntime({ Participant1:Participant, Participant2:Participant, 
    Participant3:Participant, Participant4:Participant, Participant5:Participant }))
app.listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')