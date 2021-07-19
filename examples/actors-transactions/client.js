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

const numPrtpnts = 5
const numTxns = 1

async function main () {
  let prtpnts = []
  for (let i = 1; i <= numPrtpnts; i++ ) {
    const actorName = 'Participant' + i
    const prtpnt = actor.proxy(actorName, actorName + ':id' + i)
    prtpnts.push(prtpnt)
    await actor.call(prtpnt, 'setBalance', 5000)
    console.log('Participant ', i , ': Available Balance =', await actor.call(prtpnt, 'getAvailableBalance'), 
    'Exact Balance = ', await actor.call(prtpnt, 'getExactBalance'))
  }

  let success = false
  for (let i = 0; i < numTxns; i++) {
    const txn1 = actor.proxy('Transaction', uuidv4())
    let operations = [10, 20, -10, -10, -10]
    success = await actor.call(txn1, 'transact', prtpnts, operations)
    console.log('\nTransaction success status:', success)
    await new Promise(r => setTimeout(r, 1000));
    for (let j=1; j <= prtpnts.length; j++) {
        console.log('Participant ', j , ': Available Balance =', await actor.call(prtpnts[j-1], 'getAvailableBalance'), 
        'Exact Balance = ', await actor.call(prtpnts[j-1], 'getExactBalance'))
    }
    console.log('Transaction completion status: ', await actor.call(txn1, 'txnComplete'), '\n')
    
    // Attempt to transfer more than current balance of one of the participants. 
    // That participant's prepare fails, aborting the transaction.
    const txn2 = actor.proxy('Transaction', uuidv4())
    const prtpntBalance = await actor.call(prtpnts[numPrtpnts-2], 'getAvailableBalance')
    operations = [prtpntBalance, 1, -(prtpntBalance+1), 0, 0]
    success = await actor.call(txn2, 'transact', prtpnts, operations)
    console.log('Transaction success status:', success)
    for (let j=1; j <= prtpnts.length; j++) {
        console.log('Participant ', j , ': Available Balance =', await actor.call(prtpnts[j-1], 'getAvailableBalance'), 
        'Exact Balance = ', await actor.call(prtpnts[j-1], 'getExactBalance'))
    }
    console.log('Transaction completion status: ', await actor.call(txn2, 'txnComplete'), '\n')
  }

  console.log('Terminating sidecar')
  await sys.shutdown()
}

main()
