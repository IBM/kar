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
const { actor, events, sys } = require('kar-sdk')
const cloudevents = require('cloudevents')

const app = express()

// parse arguments of service invocations
app.use(express.json({ strict: false }))

// parse events
app.use(express.json({ type: 'application/cloudevents+json' }))

app.post('/incr', (req, res) => {
  console.log('incr', req.body)
  res.json(req.body + 1)
})

app.post('/incrQuiet', (req, res) => {
  res.json(req.body + 1)
})

app.post('/shutdown', async (_reg, res) => {
  console.log('Shutting down service')
  res.sendStatus(200)
  await sys.shutdown()
  server.close(() => process.exit())
})

// example actor

class Foo {
  constructor (id) {
    this.id = id
    this.field = 42
    this.count = 0
  }

  accumulate (event) {
    this.count += event.data
  }

  async pubsub (topic) {
    const source = 'numServer'
    const type = 'number'
    await events.subscribe(this, 'accumulate', topic) // subscribe actor to topic

    // Create event 1:
    var e1 = new cloudevents.CloudEvent({
      type: type,
      source: source,
      data: 1
    })
    await events.publish(topic, e1)

    // Create event 2:
    var e2 = new cloudevents.CloudEvent({
      type: type,
      source: source,
      data: 2
    })
    await events.publish(topic, e2)

    // Create event 3:
    var e3 = new cloudevents.CloudEvent({
      type: type,
      source: source,
      data: 3
    })
    await events.publish(topic, e3)

    return 'OK'
  }

  async check (topic) {
    if (this.count >= 6) {
      await events.cancelSubscription(this, topic)
      return true
    }
    return false
  }

  activate () {
    console.log('actor', this.id, 'activate')
  }

  fail (msg) {
    console.log('actor', this.id, 'fail', msg)
    throw new Error(msg)
  }

  incr (v) {
    console.log('actor', this.id, 'incr', v)
    return v + 1
  }

  incrQuiet (v) {
    return v + 1
  }

  echo (...msgs) {
    if (msgs.length > 0) {
      for (const msg of msgs) {
        console.log(`actor ${this.id} says "${msg}"`)
      }
    } else {
      console.log(`actor ${this.id} has nothing to say`)
    }
    return 'OK'
  }

  set (key, value) {
    console.log('actor', this.id, 'set', key, value)
    actor.state.set(this, key, value)
    return 'OK'
  }

  get (key) {
    console.log('actor', this.id, 'get', key)
    return actor.state.get(this, key)
  }

  reenter (v) {
    return actor.call(this, this, 'incrQuiet', v)
  }

  deactivate () {
    console.log('actor', this.id, 'deactivate')
  }
}

app.use(sys.actorRuntime({ Foo }))

const server = app.listen(process.env.KAR_APP_PORT, '127.0.0.1')
