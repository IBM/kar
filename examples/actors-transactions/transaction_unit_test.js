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

const numPrtpnts = 4

class DummyParticipant {
  async activate () {
    this.localDecision = false
  }

  async setLocalDecision(localDecision) {
    this.localDecision = localDecision
  }

  async prepare(txnId, dummyOp) {
    // console.log('Received prepare for ', txnId, ' with op ', dummyOp)
    return this.localDecision
  }

  async commit(txnId, decision, dummyOp) {
    // console.log('Received commit for ', txnId, ' with decision ', decision)
    return
  }
}

class RaiseErrorDummyParticipant {
  async activate () {
    this.localDecision = false
    this.attempt = 1
    this.txnCommitCount = 0
  }

  async setLocalDecision(localDecision) {
    this.localDecision = localDecision
  }

  async getCommitCount() {
    return this.txnCommitCount
  }

  async prepare(txnId, dummyOp) {
    throw new Error('Raising error')
  }

  async commit(txnId, decision, dummyOp) {
    this.txnCommitCount += 1
    if (this.attempt > 0) {
      this.attempt -= 1
      throw new Error('Raising error') 
    }
    return
  }
}

async function basicTest(prtpnts) {
  const txn1 = actor.proxy('Transaction', uuidv4())
  let dummyOps = []
  let success = false
  try {
    success = await actor.call(txn1, 'transact', prtpnts, dummyOps)
  } catch(error) {
    console.assert(error.toString() == "Error: Length of participants and of operations do not match." +
    "Please ensure they have a 1:1 mapping", "Error thrown doesn't match expected error.")
  }
  
  for (let i = 0; i < prtpnts.length; i++) { dummyOps.push(i) }
  // Success case: set all dummyParticipants local decision to true and call transact.
  let localDecision = true
  for (let i = 0; i < prtpnts.length; i++) {
    await actor.call(prtpnts[i], 'setLocalDecision', localDecision)
  }
  success = await actor.call(txn1, 'transact', prtpnts, dummyOps)
  console.assert(success == true, "Transaction should succeed if all participants " + 
                                  "vote true.")
  console.log("Success scenario complete.")
  
  // Failure case: set at least one dummyParticipant's local decision to false and call transact.
  // Txn should abort.
  await actor.call(prtpnts[0], 'setLocalDecision', false)
  const txn2 = actor.proxy('Transaction', uuidv4())
  success = await actor.call(txn2, 'transact', prtpnts, dummyOps)
  console.assert(success == false, "Transaction should fail if at least one participant " + 
                                  "voted false.")
  console.log("Failure scenario complete.")
}

async function raiseErrorTest(prtpnts) {
  // If a participant raises error during prepare, the txn is aborted.
  let errorPrtpnt = actor.proxy('RaiseErrorDummyParticipant', uuidv4())
  prtpnts.push(errorPrtpnt)
  let dummyOps = []
  for (let i = 0; i < prtpnts.length; i++) { dummyOps.push(i) }

  let localDecision = true
  for (let i = 0; i < prtpnts.length; i++) {
    await actor.call(prtpnts[i], 'setLocalDecision', localDecision)
  }
  const txn1 = actor.proxy('Transaction', uuidv4())
  success = await actor.call(txn1, 'transact', prtpnts, dummyOps)
  console.assert(success == false, "Transaction should fail if at least one participant " + 
                                  "raises an error in prepare phase.")
  console.log("Raise error in prepare scenario complete.")

  // The commit of RaiseErrorDummyParticipant is designed to raise error at least
  // once. The Transaction actor needs to retry commit until it succeeds. Check if the
  // number of times commit attempted is > 0. Also check the commitComplete status.
  await new Promise(resolve => setTimeout(resolve, 1000));
  let commitCount = await actor.call(errorPrtpnt, 'getCommitCount', true)
  console.assert(commitCount > 0, "Commit count must be greater than 0 when first commit" +
                                  "throws error.")
  console.log("Raise error in commit scenario complete.")
}

async function main () {
  let prtpnts = []
  for (let i = 1; i <= numPrtpnts; i++ ) {
    const prtpnt = actor.proxy('DummyParticipant', uuidv4())
    prtpnts.push(prtpnt)
  }
  await basicTest(prtpnts)
  await raiseErrorTest(prtpnts)

  await new Promise(resolve => setTimeout(resolve, 10000));
  console.log('Terminating sidecar')
  await sys.shutdown()  
  process.exit()
}

// Server setup: register actors with KAR and start express
const app = express()
app.use(sys.actorRuntime({ DummyParticipant, RaiseErrorDummyParticipant }))
app.listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')

// Start testing
main()
