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

class TransactionParticipant {
  async activate () {
    /* The two data structures 'preparedTxns' and 'committedTxns' are managed by 
    the GenericParticipant class. Any other application related data is managed by
    the application specific participant. */
    const that = await actor.state.getAll(this)
    this.preparedTxns = that.preparedTxns || {}
    this.committedTxns = that.committedTxns || {}
    return that
  }

  async createVal(val) {
    /* Each writable class field has a rw field to indicate if a txn is writing this field;
    and a read-only array ro to indicate the list of txns only reading this field. */
    return { val: val, rw:null, ro:[] }
  }

  async returnNull() {
    return
  }

/* Non-transaction get and put methods. These do not provide ACID gurantees. Used for testing purposes. */
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

  async getAll() {
    return await actor.state.getAll(this)
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
/* End of non-transaction get and put methods. */

  isRWField(key) {
    /* Checks if a given field is a read-write field. */
    return (this[key].constructor == Object && 'val' in this[key])
  }

  async prepare(txnId, keys) {
    /* A generic prepare most commonly used by actors. This prepare is only for read-write txns. */
    let localDecision = await this.isTxnAlreadyPrepared(txnId)
    if (localDecision != null) { /* This txn is already prepared. */ return localDecision }
    localDecision = this.checkForConflictRW(txnId, keys)
    const maps = this.createPrepareValueAndWriteMap(localDecision, keys)
    await this.writePrepared(txnId, localDecision, maps.writeMap)
    return maps.values
  }

  async isTxnAlreadyPrepared(txnId) {
    /* Check if this txn is already prepared. Prepare value can be
    either true or false. If not prepared, return null. */
    if (verbose) { console.log(`${this.kar.id} received prepare for txn ${txnId}`) }
    this.preparedTxns = await actor.state.get(this, 'preparedTxns') || {}
    if (txnId in this.preparedTxns) {
      return this.preparedTxns[txnId]
    }
    return null
  }

  checkForConflictRW(txnId, keys) {
    /* Check if any key in the txn is being read or written by any other txn. If no
    other txn is accessing these keys, update the metadata of the keys. */
    let localDecision = true
    for (const i in keys) {
      const key = keys[i]
      if (this.isRWField(key)) {
        if(! (this[key].rw == null && this[key].ro.length == 0)) {
          localDecision = false }
      }
    }
    if (localDecision) {
      for (const i in keys) {
        const key = keys[i]
        if (this.isRWField(key)) {
          this[key].rw = txnId }
      }
    }
    return localDecision
  }

  checkForConflictRO(txnId, keys) {
    /* Check if any key in the txn is being written by any other txn. If no
    other txn is writing these keys, update the metadata of the keys. */
    let localDecision = true
    for (const i in keys) {
      const key = keys[i]
      if (this.isRWField(key)) {
        if (! (this[key].rw == null)) { // Some other txn is writing this field
          localDecision = false }
      } 
    }
    if (localDecision) {
      for (const i in keys) {
        const key = keys[i]
        if (this.isRWField(key)) {
            this[key].ro.push(txnId) }
      }
    }
    return localDecision
  }

  createPrepareValueAndWriteMap(localDecision, keys) {
    /* Since prepares are also reads, create a map to return to the caller.
    Create a map to write to Redis consisting of any updated metadata of keys. */
    let values = {}, writeMap = {}
    if (!localDecision) { return {values, writeMap} }
    for (const i in keys) {
      const key = keys[i]
      if (this.isRWField(key)) {
        writeMap[key] = this[key]
        values[key] = this[key].val }
      else { values[key] = this[key] }
    }
    values.vote = localDecision
    return {values, writeMap}
  }

  async writePrepared(txnId, prepared, dataMap) {
    /* Write 'preparedTxns' along with txn specific data atomically. */
    this.preparedTxns[txnId] = prepared
    const mapToWrite = Object.assign({ preparedTxns: this.preparedTxns }, dataMap)
    await actor.state.setMultiple(this, mapToWrite)
  }

  async commit(txnId, decision, update) {
    let continueCommit = await this.isTxnAlreadyCommitted(txnId)
    if (!continueCommit) { /* This txn is already committed or not prepared. */ return }
    const writeMap = this.createCommitWriteMap(txnId, decision, update)
    this.writeCommit(txnId, decision, writeMap)
    if (verbose) { console.log(`${this.kar.id} committed transaction ${txnId}.\n`) }
    return
  }

  async isTxnAlreadyCommitted(txnId) {
    /* Check if txn is already committed or not. Retrun false if txn is already committed or txn is not 
    prepared, indicating this call to commit failed. */
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

  createCommitWriteMap(txnId, decision, update) {
    /* Update the values of the fields based on txn decision. If the txn was prepared,
    release 'locks' held by this txn of any data field. */
    let writeMap = {}
    if (decision) {
      for (let key in update) {
        if (this[key] != null && this.isRWField(key) ) {
          this[key].val =  update[key] }
        else { 
          this[key] =  update[key] }
        writeMap[key] = this[key]
      }
    }
    if (this.preparedTxns[txnId]) {
      for (let i in Object.keys(this)) {
        const key = Object.keys(this)[i]
        if (this[key] != null && this.isRWField(key)) {
          if (this[key].rw == txnId) {
            this[key].rw =  null }
          if (this[key].ro.includes(txnId)) {
            this[key].ro = this[key].ro.filter(item => item !== txnId)
          }
          writeMap[key] = this[key]
        }
      }
    }
    return writeMap
  }

  async writeCommit(txnId, decision, dataMap) {
    /* Write 'committedTxns' along with txn specific data atomically. */
    this.committedTxns[txnId] = decision
    const mapToWrite = Object.assign({ committedTxns: this.committedTxns }, dataMap)
    await actor.state.setMultiple(this, mapToWrite)
  }

  async purgeTxnRecord(txnId) {
    delete this.preparedTxns[txnId]
    delete this.committedTxns[txnId]
    await actor.state.setMultiple(this, { preparedTxns: this.preparedTxns, committedTxns: this.committedTxns })
  }
}

exports.TransactionParticipant = TransactionParticipant