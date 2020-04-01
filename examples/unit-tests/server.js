const express = require('express')
const { logger, jsonParser, errorHandler, shutdown, actorRuntime } = require('kar')

const app = express()

app.use(logger, jsonParser) // enable kar logging and parsing

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
  await shutdown()
  server.close(() => process.exit())
})

// example actor

class Foo {
  constructor (id) {
    this.id = id
    this.field = 42
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

  echo (body) {
    if (body && body.msg) {
      console.log(`actor ${this.id} says "${body.msg}"`)
    } else {
      console.log(`actor ${this.id} has nothing to say`)
    }
    return 'OK'
  }

  set ({ key, value }) {
    console.log('actor', this.id, 'set', key, value)
    this.sys.set(key, value)
    return 'OK'
  }

  get ({ key }) {
    console.log('actor', this.id, 'get', key)
    return this.sys.get(key)
  }

  deactivate () {
    console.log('actor', this.id, 'deactivate')
  }
}

app.use(actorRuntime({ Foo }))

app.use(errorHandler) // enable kar error handling

const server = app.listen(process.env.KAR_APP_PORT, '127.0.0.1')
