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

const sleep = (milliseconds) => {
  return new Promise(resolve => setTimeout(resolve, milliseconds))
}

async function main () {
  let success = false
  const acct1 = actor.proxy('Account1', "123")
  await actor.call(acct1, 'setBalance', 5000)
  console.log('Account 1: Available Balance =', await actor.call(acct1, 'getAvailableBalance'), 
  'Exact Balance = ', await actor.call(acct1, 'getExactBalance'))

  const acct2 = actor.proxy('Account2', "456")
  await actor.call(acct2, 'setBalance', 5000)
  console.log('Account 2: Available Balance =', await actor.call(acct2, 'getAvailableBalance'), 
  'Exact Balance = ', await actor.call(acct2, 'getExactBalance'), '\n')

  for (let i = 0; i < 1; i++) {
    const acct1Balance = await actor.call(acct1, 'getAvailableBalance')
    const txn1 = actor.proxy('Transaction', uuidv4())
    success = await actor.tell(txn1, 'transfer', acct1, acct2, 15)
    const txn2 = actor.proxy('Transaction', uuidv4())
    success = await actor.tell(txn2, 'transfer', acct1, acct2, 20)
    console.log('Transaction success status:', success)
    console.log('Account 1: Available Balance =', await actor.call(acct1, 'getAvailableBalance'), 
    'Exact Balance = ', await actor.call(acct1, 'getExactBalance'))
    console.log('Account 2: Available Balance =', await actor.call(acct2, 'getAvailableBalance'), 
    'Exact Balance = ', await actor.call(acct2, 'getExactBalance'))
    console.log('Transaction completion status: ', await actor.call(txn1, 'txnComplete'), '\n')
    
    // Attempt to transfer more than current balance of acct2. 
    // acct2 updateBalance fails, reverting acct1's transfer.
    const txn3 = actor.proxy('Transaction', uuidv4())
    const acct2Balance = await actor.call(acct2, 'getAvailableBalance')
    success = await actor.call(txn3, 'transfer', acct1, acct2, -(acct2Balance+1))
    console.log('Transaction success status:', success)
    console.log('Account 1: Available Balance =', await actor.call(acct1, 'getAvailableBalance'), 
    'Exact Balance = ', await actor.call(acct1, 'getExactBalance'))
    console.log('Account 2: Available Balance =', await actor.call(acct2, 'getAvailableBalance'), 
    'Exact Balance = ', await actor.call(acct2, 'getExactBalance'))
    console.log('Transaction completion status: ', await actor.call(txn3, 'txnComplete'), '\n')
  }

  console.log('Terminating sidecar')
  await sys.shutdown()
}

main()
