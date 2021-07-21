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
    console.log(`Received prepare for txn ${txnId}`)
    this.preparedTxns = await actor.state.get(this, 'preparedTxns') || {}
    if (txnId in this.preparedTxns) {
      return this.preparedTxns[txnId]
    }
    return null
  }

  async writePrepared(txnId, prepared, dataMap) {
    /* Write 'preparedTxns' along with application specific data atomically. */
    this.preparedTxns[txnId] = prepared
    dataMap['preparedTxns'] = this.preparedTxns
    await actor.state.setMultiple(this, dataMap)
  }
  
  async isTxnPreparedTrue(txnId) {
    /* Can be true or false */
    return this.preparedTxns[txnId]
  }

  async commit(txnId, decision) {
    /* Check if txn is already committed or not. Return true only if this particular
    call to commit succeeds; retrun false if txn is already committed or txn is not 
    prepared, indicating this call to commit failed. */
    if (!(txnId in this.preparedTxns)) { 
      console.log('An unprepared txn', txnId, ' cannot be committed')
      return false
    }
    console.log(`Received commit for txn ${txnId}`)
    this.committedTxns = await actor.state.get(this, 'committedTxns') || {}
    if (txnId in this.committedTxns) {
      // Already committed this txn.
      return false
    }
    console.log(`Transaction ${txnId}'s decision is`, decision)
    this.committedTxns[txnId] = decision
    return true
  }

  async writeCommit(dataMap) {
    /* Write 'committedTxns' along with application specific data atomically. */
    dataMap['committedTxns'] = this.committedTxns
    await actor.state.setMultiple(this, dataMap)
  }
}

exports.GenericParticipant = GenericParticipant