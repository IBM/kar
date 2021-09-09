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

const numPrtpnts = 3
const numTxns = 10

async function functionalTest(prtpnts) {
  let availableQuantity1 = [], exactQuantity1 = []
  for (let j=0; j < prtpnts.length; j++) {
    availableQuantity1.push(await actor.call(prtpnts[j], 'getAvailableQuantity'))
    exactQuantity1.push(await actor.call(prtpnts[j], 'getExactQuantity'))
  }
  // Success case
  const txn1 = actor.proxy('Transaction', uuidv4())
  let operations = [10, 20, 10]
  success = await actor.call(txn1, 'transact', prtpnts, operations)
  console.log('\nTransaction success status:', success)
  while (!await actor.call(txn1, 'txnComplete')) { }
  let availableQuantity2 = [], exactQuantity2 = []
  for (let j=0; j < prtpnts.length; j++) {
    availableQuantity2.push(await actor.call(prtpnts[j], 'getAvailableQuantity'))
    exactQuantity2.push(await actor.call(prtpnts[j], 'getExactQuantity'))
    console.assert(availableQuantity2[j] == (availableQuantity1[j] - operations[j]),
                  'The available quantity of Item', j, 'after txn completion is incorrect.')
    console.assert(exactQuantity2[j] == (exactQuantity1[j] - operations[j]),
                  'The exact quantity of Item', j, 'after txn completion is incorrect.')
  }
  console.log('Transaction completion status: ', await actor.call(txn1, 'txnComplete'), '\n')

  // Attempt to transfer more than current available quantity for one of the participants. 
  // That participant's prepare fails, aborting the transaction.
  const txn2 = actor.proxy('Transaction', uuidv4())
  const prtpntQuantity = await actor.call(prtpnts[numPrtpnts-1], 'getAvailableQuantity')
  operations = [10, 10, (prtpntQuantity+1)]
  success = await actor.call(txn2, 'transact', prtpnts, operations)
  console.log('Transaction success status:', success)
  while (!await actor.call(txn2, 'txnComplete')) { }
  let availableQ, exactQ
  for (let j=0; j < prtpnts.length; j++) {
    availableQ = await actor.call(prtpnts[j], 'getAvailableQuantity')
    exactQ = await actor.call(prtpnts[j], 'getExactQuantity')
    console.assert(availableQ == availableQuantity2[j],
                  'The available quantity of Item', j, 'after unsuccessful txn is incorrect.')
    console.assert(exactQ == exactQuantity2[j],
                  'The exact quantity of Item', j, 'after unsuccessful txn is incorrect.')
  }
  console.log('Transaction completion status: ', await actor.call(txn2, 'txnComplete'), '\n')
  console.log('Functional test complete.\n')
}

async function loadTest(prtpnts) {
  let success = false
  for (let i = 0; i < numTxns; i++) {
    const txn1 = actor.proxy('Transaction', uuidv4())
    let operations = [10, 20, 10]
    success = await actor.call(txn1, 'transact', prtpnts, operations)
    console.log('\nTransaction success status:', success)  
    console.log('Transaction completion status: ', await actor.call(txn1, 'txnComplete'), '\n')
  }
  for (let j=1; j <= prtpnts.length; j++) {
    console.log('Item ', j , ': Available Quantity =', await actor.call(prtpnts[j-1], 'getAvailableQuantity'), 
    'Exact Quantity = ', await actor.call(prtpnts[j-1], 'getExactQuantity'))
  }
}

async function main () {
  let prtpnts = []
  for (let i = 1; i <= numPrtpnts; i++ ) {
    const actorName = 'Item' + i
    const prtpnt = actor.proxy(actorName, actorName + ':id' + i)
    prtpnts.push(prtpnt)
    await actor.call(prtpnt, 'setQuantity', 5000)
  }
  await functionalTest(prtpnts)
  await loadTest(prtpnts)

  console.log('Terminating sidecar')
  await sys.shutdown()
}

main()