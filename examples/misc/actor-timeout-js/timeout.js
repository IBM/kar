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
const { actor, sys } = require('kar')

const app = express()

class Test {
  async A () {
    console.log('entering method A')
    await actor.call(this, this, 'B') // synchronous call to self within the same session -> OK
    console.log('exiting method A')
  }

  async B () {
    console.log('entering method B')
    await actor.call(this, 'A') // synchronous call to self in a new session -> deadlock
    console.log('exiting method B')
  }
}

app.use(sys.actorRuntime({ Test }))

app.listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')

async function main () {
  try {
    await actor.call(actor.proxy('Test', '123'), 'A')
  } catch (err) {
    console.error(err)
  }
}

main()
