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
var generic_participant = require('./generic_participant.js')

class GenericParticipantActor extends generic_participant.GenericParticipant {
}

async function prepareTest(gp) {
  // Call prepare + writePrepare on same txn twice. Second time should be non-null.
  const txnId = '123'
  const localDecision = true
  const storedDecision1 = await actor.call(gp, 'prepare', txnId)
  if(storedDecision1 == null) { await actor.call(gp, 'writePrepared', txnId, localDecision, {}) }
  console.assert(storedDecision1 == null, "First invocation of 'prepare' for a new txn should return null.")

  const storedDecision2 = await actor.call(gp, 'getTxnLocalDecision', txnId)
  console.assert(storedDecision2 == localDecision, "Stored local decision must be the same "+
                "as what was sent.")

  const storedDecision3 = await actor.call(gp, 'prepare', txnId)
  console.assert(storedDecision3 == localDecision, "Repeated invocation of 'prepare' for a" +
            "given txn should return same non-null value as the set local decision.")
}

async function commitTest(gp) {
  const txnId1 = '456'
  const localDecision = true
  // An un-prepared txn cannot be committed; commit invocation must return false.
  const continueCommit1 = await actor.call(gp, 'commit', txnId1, localDecision)
  if (continueCommit1) { await actor.call(gp, 'writeCommit', txnId1, localDecision, {}) }
  console.assert(continueCommit1 == false, "Commit invocation with an unprepared txn " +
                                           "should not succeed.")

  // Calling prepare after commit should return false, which will be no-op for the application.
  const storedDecision1 = await actor.call(gp, 'prepare', txnId1)
  console.assert(storedDecision1 == false, "Calling prepare after calling commit for a txn " + 
                                    "should return false.")

  // Prepare txn and then call commit + writeCommit on same txn twice. Second time should be false.
  const txnId2 = '789'
  const storedDecision2 = await actor.call(gp, 'prepare', txnId2)
  if (storedDecision2 == null) { await actor.call(gp, 'writePrepared', txnId2, localDecision, {}) }
  const continueCommit2 = await actor.call(gp, 'commit', txnId2, localDecision)
  console.assert(continueCommit2 == true, "First invocation of 'commit' for a new and " + 
                                          "prepared txn should return true.")
  await actor.call(gp, 'writeCommit', txnId2, localDecision, {})

  const continueCommit3 = await actor.call(gp, 'commit', txnId2, localDecision)
  console.assert(continueCommit3 == false, "Repeated invocation of 'commit' for a" +
                                           "committed txn should return false.")
}

async function main () {
  const gp = actor.proxy('GenericParticipantActor', uuidv4())
  await prepareTest(gp)
  await commitTest(gp)

  console.log('Terminating sidecar')
  await sys.shutdown()
  process.exit()
}

// Server setup: register actors with KAR and start express
const app = express()
app.use(sys.actorRuntime({ GenericParticipantActor }))
app.listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')

// Start testing
main()
