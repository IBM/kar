/*
 * Copyright IBM Corporation 2020,2023
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
const t = require('../txn_framework/transaction.js')
const tp = require('../txn_framework/txn_participant.js')
const verbose = process.env.VERBOSE

class MoneyTransfer extends t.Transaction {
  async activate () {
    await super.activate()
  }

  async prepareTxn (accts, operations) {
    const actorUpdates = {}; let decision = true
    for (const i in accts) {
      const acct = actor.proxy('Account', accts[i])
      actorUpdates[accts[i]] = { actr: acct, update: operations[i] }
    }
    await actor.state.set(this, 'actorUpdates', actorUpdates)
    for (const i in actorUpdates) {
      actorUpdates[i].values = await actor.asyncCall(actorUpdates[i].actr, 'prepare', this.txnId, actorUpdates[i].update)
    }
    for (const i in actorUpdates) { actorUpdates[i].values = await actorUpdates[i].values() }
    for (const i in actorUpdates) { decision = decision && actorUpdates[i].values.vote }
    await actor.state.setMultiple(this, { decision: decision, actorUpdates: actorUpdates })
    return decision
  }

  async startTxn (accts, operations) {
    if (verbose) { console.log(`Begin transaction ${this.txnId}.`) }
    const that = await actor.state.getAll(this)
    if (that.commitComplete) { return that.decision }
    let decision = that.decision
    if (that.decision == null) {
      try {
        decision = await this.prepareTxn(accts, operations)
      } catch (error) {
        console.log(error.toString())
        // If decision is not already set, abort this txn as something went wrong.
        if (await actor.state.get(this, 'decision') == null) { decision = false }
      }
    }
    await actor.tell(this, 'sendCommitAsync', decision)
    return decision
  }
}

class Account extends tp.TransactionParticipant {
  async activate () {
    const that = await super.activate()
    this.exactBalance = that.exactBalance || 5000
    this.credits = that.credits || 0
    this.debits = that.debits || 0
    this.preparedTxns = that.preparedTxns || {}
    this.committedTxns = that.committedTxns || {}
  }

  async setBalance (balance) {
    this.exactBalance = balance
    actor.state.set(this, 'exactBalance', this.exactBalance)
  }

  async getAvailableBalance () {
    return this.exactBalance - this.debits
  }

  async getExactBalance () {
    return this.exactBalance
  }

  async prepare (txnId, amt) {
    let localDecision = await super.isTxnAlreadyPrepared(txnId)
    if (localDecision != null) { /* This txn is already prepared. */ return localDecision }
    localDecision = (this.exactBalance - this.debits + amt > 0)
    if (localDecision) { amt < 0 ? this.debits -= amt : this.credits += amt }
    await super.writePrepared(txnId, localDecision, { debits: this.debits, credits: this.credits })
    return { vote: localDecision }
  }

  async commit (txnId, decision, amt) {
    const continueCommit = await super.isTxnAlreadyCommitted(txnId)
    if (!continueCommit) { /* This txn is already committed or not prepared. */ return }
    if (decision) { this.exactBalance += amt }
    if (super.isLocalDecisionTrue(txnId)) { amt < 0 ? this.debits += amt : this.credits -= amt }
    this.writeCommit(txnId, decision, { exactBalance: this.exactBalance, debits: this.debits, credits: this.credits })
    if (verbose) { console.log(`${this.kar.id} committed transaction ${txnId}.\n`) }
  }
}

// Server setup: register actors with KAR and start express
const app = express()
app.use(sys.actorRuntime({ MoneyTransfer, Account }))
sys.h2c(app).listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')
