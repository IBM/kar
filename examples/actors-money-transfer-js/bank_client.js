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

async function testTermination (success) {
  if (!success) {
    console.log('FAILED; setting non-zero exit code')
    process.exitCode = 0
  } else {
    console.log('SUCCESS')
    process.exitCode = 1
  }

  console.log('Terminating sidecar')
  await sys.shutdown()
}

async function main () {
  let success = false
  const acct1 = actor.proxy('Account1', "123")
  console.log(await actor.call(acct1, 'getBalance'))

  const acct2 = actor.proxy('Account2', "456")
  console.log(await actor.call(acct2, 'getBalance'))

  const txn1 = actor.proxy('Transaction', '1234')
  success = await actor.call(txn1, 'transfer', acct1, acct2, 40)
  console.log(await actor.call(acct1, 'getBalance'))
  console.log(await actor.call(acct2, 'getBalance'))

  // Re-executing a transaction does not change the balance
  const txn2 = actor.proxy('Transaction', '1234')
  success = await actor.call(txn2, 'transfer', acct1, acct2, 40)
  console.log(await actor.call(acct1, 'getBalance'))
  console.log(await actor.call(acct2, 'getBalance'))

  // Attempt to transfer more than current balance of acct2. 
  // acct2 updateBalance fails, reverting acct1's transfer.
  const txn3 = actor.proxy('Transaction')
  const acct2Balance = await actor.call(acct2, 'getBalance')
  success = await actor.call(txn3, 'transfer', acct1, acct2, -(acct2Balance+1))
  console.log(await actor.call(acct1, 'getBalance'))
  console.log(await actor.call(acct2, 'getBalance'))

  console.log('Transaction success status:', success)
  testTermination(success)
}

main()
