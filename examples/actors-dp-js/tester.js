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

const sleep = (milliseconds) => {
  return new Promise(resolve => setTimeout(resolve, milliseconds))
}

async function testTermination (failure) {
  if (failure) {
    console.log('FAILED; setting non-zero exit code')
    process.exitCode = 1
  } else {
    console.log('SUCCESS')
    process.exitCode = 0
  }

  console.log('Terminating sidecar')
  await sys.shutdown()
}

async function main () {
  let failure = false
  let countdown = 60
  const cafe = actor.proxy('Cafe', 'Cafe de Flore')

  console.log('Serving a meal:')
  const table = await actor.call(cafe, 'seatTable', 20, 5)

  let occupancy = 1
  while (occupancy > 0 & !failure) {
    occupancy = await actor.call(cafe, 'occupancy', table)
    console.log(`Table occupancy is ${occupancy}`)
    await sleep(2000)
    countdown = countdown - 1
    if (countdown < 0) {
      console.log('TOO SLOW: Countdown reached 0!')
      failure = true
    }
  }

  testTermination(failure)
}

main()
