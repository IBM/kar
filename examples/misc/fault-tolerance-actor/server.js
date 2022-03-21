/*
 * Copyright IBM Corporation 2020,2022
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

const app = express()

// parse arguments of service invocations
app.use(express.json({ strict: false }))

// example actor

class A {
  async f (v) {
    console.log('>', this.kar)
    var result = await actor.call(this, actor.proxy('B', '123'), 'g', v)
    console.log('<', this.kar)
    return result
  }
}

class B {
  async g (v) {
    console.log('>', this.kar)
    await new Promise(r => setTimeout(r, 15000))
    console.log('<', this.kar)
    return v + 1
  }
}

app.use(sys.actorRuntime({ A, B }))

app.listen(process.env.KAR_APP_PORT, '127.0.0.1')
