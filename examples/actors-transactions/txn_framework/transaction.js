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

const { actor } = require('kar-sdk')
const { v4: uuidv4 } = require('uuid')

class Transaction {
  async activate () {
    this.txnId = await actor.state.get(this, 'txnId') || uuidv4()
    await actor.state.set(this, 'txnId', this.txnId)
  }

  async txnComplete () {
    return await actor.state.get(this, 'commitComplete')
  }

  async prepareTxn (actorUpdates, prepareFunc = 'prepare') {
    /* actorUpdates is a map of thr form { 'actorName': { actr: <actor instance> } }. And prepareFunc
    is txn specific prepare method with default 'prepare' method. This method parallely invokes
    prepare of each actor of actorUpdates and fills in the values for each actor. The output is of
    the form { 'actorName': { actr: <actor instance>, values: <values returned by prepare> } } */
    await actor.state.set(this, 'actorUpdates', actorUpdates)
    for (const i in actorUpdates) {
      actorUpdates[i].values = await actor.asyncCall(actorUpdates[i].actr, prepareFunc, this.txnId)
    }
    for (const i in actorUpdates) { actorUpdates[i].values = await actorUpdates[i].values() }
    return actorUpdates
  }

  async sendCommitAsync (decision, commitFunc = 'commit') {
    /* This method assumes the necessary actors and their updates are stored in Redis. It expects actorUpdates
    to be a map of the form { 'actorName': { actr: <actor instance>, updated: <key-value updates> } }.
    It parallely calls commitFunc commit method specified by the caller (or 'commit' by default). When all calls
    return, it sets 'commitComplete' and purges txn record on all participating actors. */
    const getVals = await Promise.all([actor.state.get(this, 'commitComplete'), actor.state.get(this, 'actorUpdates')])
    if (getVals[0]) { return }
    const actorUpdates = getVals[1]
    try {
      const done = []
      for (const i in actorUpdates) {
        done.push(await actor.asyncCall(actorUpdates[i].actr, commitFunc, this.txnId, decision, actorUpdates[i].update))
      }
      for (const i in done) { await done[i]() }
      await actor.state.set(this, 'commitComplete', true)
    } catch (error) {
      console.log(error.toString())
      return this.sendCommitAsync(decision)
    }
    await actor.tell(this, 'purgeTxn')
  }

  async purgeTxn () {
    const getVals = await Promise.all([actor.state.get(this, 'commitComplete'), actor.state.get(this, 'actorUpdates')])
    if (getVals[0]) {
      const actorUpdates = getVals[1]
      for (const i in actorUpdates) {
        await actor.call(actorUpdates[i].actr, 'purgeTxnRecord', this.txnId)
      }
    } else (await this.purgeTxn())
  }
}

exports.Transaction = Transaction
