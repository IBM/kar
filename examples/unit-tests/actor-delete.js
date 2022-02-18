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

const { actor, sys } = require('kar-sdk')
const express = require('express')
const app = express()

class Victim {
  constructor (id) {
    this.id = id
  }

  async activate () {
    console.log('actor', this.id, 'activate')
    const state = await actor.state.getAll(this)
    this.timestamp = state.timestamp || Date.now()
  }

  async deactivate () {
    console.log('actor', this.id, 'deactivate')
    await this.checkpoint()
  }

  getStamp () {
    return this.timestamp
  }

  async checkpoint () {
    console.log(`checkpoint invoked ${this.timestamp}`)
    await actor.state.set(this, 'timestamp', this.timestamp)
  }

  async deleteMyself () {
    await actor.remove(this)
  }

  async deleteMyState () {
    await actor.state.removeAll(this)
  }

  async deleteOther (id) {
    await actor.remove(actor.proxy('Victim', id))
  }
}

app.use(sys.actorRuntime({ Victim }))
app.listen(process.env.KAR_APP_PORT, '127.0.0.1')
