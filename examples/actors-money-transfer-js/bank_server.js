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

  async txnComplete() {
    return await actor.state.get(this, 'txnComplete')
  }

  async transfer (sender, receiver, amt) {
    console.log(`Begin transaction ${this.txnId}.`)
    await actor.state.set(this, 'txnId', this.txnId)
    const that = await actor.state.getAll(this)
    if (that.txnComplete) {
      return that.decision
    }
    let decision = that.decision
    if (that.decision == null) { 
      try {
        const vote1 = await actor.asyncCall(sender, 'prepare', this.txnId, amt)
        const vote2 = await actor.asyncCall(receiver, 'prepare', this.txnId, -amt)
        decision = await vote1() && await vote2()
        await actor.state.set(this, 'decision', decision)
      } catch (error) {
        // If decision is not already set, abort this txn as something went wrong.
        if (await actor.state.get(this, 'decision') == null) {
          decision = false
        }
      }
    }
    if (that.commitIssued == null) {
      try {
        const done1 = await actor.asyncCall(sender, 'commit', this.txnId, decision, amt)
        const done2 = await actor.asyncCall(receiver, 'commit', this.txnId, decision, -amt)
        await actor.tell(this, 'setTxnCompleteAsync', done1(), done2())
        await actor.state.set(this, 'commitIssued', true)
      } catch (error) {
        console.log(error.toString())
        return this.transfer(sender, receiver, amt)
      }
    }
    console.log(`End transaction ${this.txnId}.\n`)
    return decision 
  }

  async setTxnCompleteAsync(done1, done2) {
    try {
      await done1 && await done2
      await actor.state.set(this, 'txnComplete', true)
    } catch (error) {
      console.log(error.toString())
      return this.transfer(sender, receiver, amt)
    }    
  }
}

class Account {
  async activate () {
    const that = await actor.state.getAll(this)
    this.exactBalance = that.exactBalance || 1000
    this.credits = that.credits || 0
    this.debits =  that.debits || 0
    this.preparedTxns = that.preparedTxns || {}
    this.committedTxns = that.committedTxns || {}
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
    console.log('Debits: ', this.debits, " Credits: ", this.credits)
    amt > 0? this.debits += amt : this.credits -= amt
    console.log('Debits: ', this.debits, " Credits: ", this.credits)
    const prepared = await this.getAvailableBalance() > 0? true : false
    this.preparedTxns[txnId] = prepared
    await actor.state.setMultiple(this, {debits: this.debits, credits: this.credits,
      preparedTxns: this.preparedTxns})
    return prepared
  }
  
  async commit(txnId, decision, amt) {
    console.log(`Received commit for txn ${txnId}`)
    this.committedTxns = await actor.state.get(this, 'committedTxns') || {}
    if (txnId in this.committedTxns) {
      // Already committed this txn.
      return this.committedTxns[txnId]
    } else if (!(txnId in this.preparedTxns)) {
      // This txn wasn't prepared, so it cannot be committed.
      return false
    }
    console.log(`Transaction ${txnId}'s decision is`, decision)
    this.committedTxns[txnId] = decision
    if (decision == true) {
      this.exactBalance -= amt
    }
    amt > 0? this.debits -= amt : this.credits += amt
    await actor.state.setMultiple(this, {exactBalance: this.exactBalance, debits: this.debits,
      credits: this.credits, committedTxns : this.committedTxns})
    console.log(`Committed transaction ${txnId}. Exact balance is ${this.exactBalance}.\n`)
    return decision
  }
}

// Server setup: register actors with KAR and start express
const app = express()
app.use(sys.actorRuntime({ Account1:Account, Account2:Account, Transaction }))
app.listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')