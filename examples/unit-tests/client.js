/*
 * Copyright IBM Corporation 2020,2023
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

const { call, asyncCall, actor } = require('kar-sdk')

async function main () {
  // synchronous call
  console.log(await call('myService', 'incr', 42))

  // async call 1
  const f = await asyncCall('myService', 'incr', 22)

  // async call 2
  const f2 = await actor.asyncCall(actor.proxy('Foo', 123), 'incr', 42)

  // await callback 1
  console.log(await f())

  // await callback
  console.log(await f2())
}

main()
