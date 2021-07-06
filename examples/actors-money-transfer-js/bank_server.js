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

  async raiseTimeoutErr() {
    throw 'Timeout: One or more actors not responding.'
  }

  async rollBack(txnId, sender, receiver, amt) {
    console.log(`Rolling back txn ${txnId}`)
    actor.tell(sender, 'updateBalance', txnId+'-abort', -amt)
    actor.tell(receiver, 'updateBalance', txnId+'-abort', amt)
  }

  async transfer (sender, receiver, amt) {
    try {
      if (await actor.call(sender, 'updateBalance', this.txnId, amt)) {
        // setTimeout(this.raiseTimeoutErr, 2000);
        if (await actor.call(receiver, 'updateBalance', this.txnId, -amt)) {
          return true
        }
        // The second transfer did not succeed; rollback the first transfer.
        await actor.call(sender, 'updateBalance', this.txnId+'-abort', -amt)
        return false
      }
    } catch(error) {
      console.log(error.toString())
      if (error.toString().includes('timeout')) {
        console.log("Caught error. Calling rollback")
        // One or more actors are down. Revert the transactions.
        this.rollBack(this.txnId, sender, receiver, amt)
      }
    }
    return false
  }
}

class Account {
  async activate () {
    this.currBalance = await actor.state.get(this, 'currBalance') || 1000
    this.txnIds = await actor.state.get(this, 'txnIds') || []
    this.ignoreTxnIds =  await actor.state.get(this, 'ignoreTxnIds') || []
  }

  async getBalance() {
    return this.currBalance
  }

  async updateBalance (txnId, amt) {
    const that = await actor.state.getAll(this)
    this.txnIds = that.txnIds || []
    this.ignoreTxnIds = that.ignoreTxnIds || []
  
    console.log(`Received txn ${txnId}`)
    if (this.txnIds.includes(txnId)) {
      // This account has already executed this transaction.
      return true
    } else if (txnId.endsWith('abort')) {
      // This is a revert transaction; check if forward transaction
      // is already executed - if no, remeber forward transaction id
      // so that it can be ignored. Otherwise, execute reversal txn.
      const originalTxnId = txnId.substring(0, txnId.length-6)
      if (!this.txnIds.includes(originalTxnId)) {
        // Received abort txn before original txn; add original txn to ignore list.
        this.ignoreTxnIds.push(originalTxnId)
        await actor.state.setMultiple(this, {ignoreTxnIds: this.ignoreTxnIds, txnIds: this.txnIds})
        return false
      }
    } else if (this.ignoreTxnIds.includes(txnId)) {
      return false
    }
    let success = false
    if ((this.currBalance - amt) >= 0) {
      this.currBalance -= amt
      this.txnIds.push(txnId)
      await actor.state.setMultiple(this, {currBalance: this.currBalance, txnIds: this.txnIds})
      console.log(`Txn ${txnId} updated ${this.kar.id}'s balance to ${this.currBalance}.`)
      success = true
    }
    return success
  }
}

// Server setup: register actors with KAR and start express
const app = express()
app.use(sys.actorRuntime({ Account1:Account, Account2:Account, Transaction }))
app.listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')