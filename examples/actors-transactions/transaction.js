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

const { actor, sys } = require('kar-sdk')
const { v4: uuidv4 } = require('uuid')

const verbose = process.env.VERBOSE

class Transaction {
  async activate () {
    this.txnId = await actor.state.get(this, 'txnId') || uuidv4()
    await actor.state.set(this, 'txnId', this.txnId)
  }

  async txnComplete() {
    return await actor.state.get(this, 'commitComplete')
  }

  async transact (prtpnts, operations) {
    if (prtpnts.length != operations.length) {
      throw new Error('Length of participants and of operations do not match.'+
      'Please ensure they have a 1:1 mapping')
    }
    if (verbose) { console.log(`Begin transaction ${this.txnId}.`) }
    const that = await actor.state.getAll(this)
    if (that.commitComplete) {
      return that.decision
    }
    let decision = that.decision
    if (that.decision == null) {
      try {
        let votes = []
        for (const i in prtpnts) {
          votes.push(await actor.asyncCall(prtpnts[i], 'prepare', this.txnId, operations[i]))
        }
        decision = true
        for (const i in votes) { decision = decision && await votes[i]() }
        await actor.state.set(this, 'decision', decision)
      } catch (error) {
        console.log(error.toString())
        // If decision is not already set, abort this txn as something went wrong.
        if (await actor.state.get(this, 'decision') == null) {
          decision = false
        }
      }
    }
    if (that.commitComplete == null) {
      await actor.tell(this, 'sendCommitAsync', prtpnts, operations, decision)
    }
    if (verbose) { console.log(`End transaction ${this.txnId}.\n`) }
    return decision 
    
  }

  async sendCommitAsync(prtpnts, operations, decision) {
    if (await actor.state.get(this, 'commitComplete')) { return }
    try {
      let done = []
      for (const i in prtpnts) {
        done.push(await actor.asyncCall(prtpnts[i], 'commit', this.txnId, decision, operations[i]))
      }
      for (const i in done) { await done[i]() }
      await actor.state.set(this, 'commitComplete', true)
    } catch (error) {
      console.log(error.toString())
      return this.sendCommitAsync(prtpnts, operations, decision)
    }
  }
}

exports.Transaction = Transaction