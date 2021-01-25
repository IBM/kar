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

const { actor } = require('kar-sdk')

const sleep = (milliseconds) => {
  return new Promise(resolve => setTimeout(resolve, milliseconds))
}

async function main () {
  const now = Date.now()
  const targetTime = new Date(now + 3 * 1000)
  const period = '5s'
  const a22 = actor.proxy('Foo', '22')
  const a23 = actor.proxy('Foo', '23')
  const a2112 = actor.proxy('Foo', '2112')
  await actor.reminders.schedule(a22, 'echo', { id: 'ticker', targetTime, period }, 'hello', 'my friend', 'my foe')
  await actor.reminders.schedule(a23, 'echo', { id: 'ticker', targetTime, period })
  await actor.reminders.schedule(a2112, 'echo', { id: 'ticker', targetTime, period }, 'Syrinx')
  await actor.reminders.schedule(a22, 'echo', { id: 'once', targetTime }, 'carpe diem')
  console.log(await actor.reminders.get(a23))
  console.log(await actor.reminders.get(a22, 'noone'))
  console.log(await actor.reminders.get(a22, ''))
  console.log(await actor.reminders.get(a22, 'ticker'))
  await sleep(20000)
  await actor.reminders.cancel(a22, 'ticker')
  await actor.reminders.cancel(a2112)
  await sleep(20000)
}

main()
