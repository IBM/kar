const express = require('express')
const { actor, publish, subscribe, unsubscribe, sys } = require('kar')
const cloudevents = require('cloudevents-sdk/v1')

const app = express()

// pubsub test
let success
let count = 0

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

app.post('/pubsub', async (req, res) => {
  const topic = req.body
  const source = 'numServer'
  const type = 'number'
  const promise = new Promise(resolve => { success = resolve })

  await subscribe(topic, 'accumulate') // subscribe service to topic

  // Create event 1:
  const e1 = cloudevents.event()
    .type(type)
    .source(source)
    .id(1)
    .data(1)
  await publish(topic, e1)

  // Create event 2:
  const e2 = cloudevents.event()
    .type(type)
    .source(source)
    .id(2)
    .data(2)
  await publish(topic, e2)

  // Create event 3:
  const e3 = cloudevents.event()
    .type(type)
    .source(source)
    .id(3)
    .data(3)
  await publish(topic, e3)

  await promise
  await unsubscribe(req.body)
  res.sendStatus(200)
})

app.post('/accumulate', (req, res) => {
  const payload = req.body.data
  count += payload
  if (count >= 6) success()
  res.sendStatus(200)
})

// example actor

class Foo {
  constructor (id) {
    this.id = id
    this.field = 42
  }

  accumulate (e) {
    const v = e.data
    count += v
    if (count >= 6) success()
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
