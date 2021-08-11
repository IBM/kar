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

const verbose = process.env.VERBOSE

class GenericParticipant {
  async activate () {
    /* The two data structures 'preparedTxns' and 'committedTxns' are managed by 
    the GenericParticipant class. Any other application related data is managed by
    the application specific participant. */
    const that = await actor.state.getAll(this)
    this.preparedTxns = that.preparedTxns || {}
    this.committedTxns = that.committedTxns || {}
    return that
  }

  async prepare(txnId) {
    /* Check if this txn is already prepared. Prepare value can be
    either true or false. If not prepared, return null. */
    if (verbose) { console.log(`${this.kar.id} received prepare for txn ${txnId}`) }
    this.preparedTxns = await actor.state.get(this, 'preparedTxns') || {}
    if (txnId in this.preparedTxns) {
      return this.preparedTxns[txnId]
    }
    return null
  }

  async checkVersionConflict(update) {
    let localDecision = true
    for ( let key in update) {
      if (update[key].constructor == Object) {
        if (this[key].ts != update[key].ts) { localDecision = false} }
    }
    return localDecision
  }

  async createPrepareWriteMap(localDecision, update) {
    let writeMap = {}
    if (localDecision) { 
      for ( let key in update) {
        if (update[key].constructor == Object) {
          this[key].ts += 1
          writeMap[key] = this[key] } }
    }
    return writeMap
  }

  async writePrepared(txnId, prepared, dataMap) {
    /* Write 'preparedTxns' along with application specific data atomically. */
    this.preparedTxns[txnId] = prepared
    const mapToWrite = Object.assign({ preparedTxns: this.preparedTxns }, dataMap)
    await actor.state.setMultiple(this, mapToWrite)
  }
  
  async getTxnLocalDecision(txnId) {
    /* Can be true or false */
    this.preparedTxns = await actor.state.get(this, 'preparedTxns') || {}
    return this.preparedTxns[txnId]
  }

  async commit(txnId, decision) {
    /* Check if txn is already committed or not. Return true only if this particular
    call to commit succeeds; retrun false if txn is already committed or txn is not 
    prepared, indicating this call to commit failed. */
    if (verbose) { console.log(`${this.kar.id} received commit for txn ${txnId} with decision `, decision) }
    this.committedTxns = await actor.state.get(this, 'committedTxns') || {}
    if (!(txnId in this.preparedTxns)) {
      this.preparedTxns[txnId] = false
      this.committedTxns[txnId] = false
      await actor.state.setMultiple(this, {preparedTxns: this.preparedTxns,
                                    committedTxns: this.committedTxns})
      console.log('An unprepared txn', txnId, ' cannot be committed')
      return false
    }
    if (txnId in this.committedTxns) { /* Already committed this txn.*/ return false }
    return true
  }

  async createCommitWriteMap(txnId, decision, update) {
    let writeMap = {}
    if (decision) {
      for (let key in update) {
        if (update[key].constructor == Object) {
          this[key].val =  update[key].val
          this[key].ts +=  1
          writeMap[key] = this[key] } 
        else {
          this[key] =  update[key]
          writeMap[key] = this[key]
        } }
    }
    return writeMap
  }

  async writeCommit(txnId, decision, dataMap) {
    /* Write 'committedTxns' along with application specific data atomically. */
    this.committedTxns[txnId] = decision
    const mapToWrite = Object.assign({ committedTxns: this.committedTxns }, dataMap)
    await actor.state.setMultiple(this, mapToWrite)
  }

  async createVal(val) {
    return { val: val, ts:0 }
  }

  async get(key) {
    return this[key]
  }

  async getMultiple(keys) {
    let resp = {}
    for (let i in keys) {
      resp[keys[i]] = this[keys[i]]
    }
    return resp
  }

  async put(key, value) {
    this[key] = value
    await actor.state.set(this, key, value)
  }

  async putMultiple(keyValueMap) {
    for (let key in keyValueMap) {
      this[key] = keyValueMap[key]
    }
    await actor.state.setMultiple(this, keyValueMap)
  }
}

exports.GenericParticipant = GenericParticipant