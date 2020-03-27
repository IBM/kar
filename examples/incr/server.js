const express = require('express')
const { logger, jsonParser, errorHandler, shutdown } = require('kar')

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

app.get('/actor/foo/:id', (req, res) => {
  console.log('actor', req.params.id, 'activate')
  res.sendStatus(200)
})

app.delete('/actor/foo/:id', (req, res) => {
  console.log('actor', req.params.id, 'deactivate')
  res.sendStatus(200)
})

app.post('/actor/foo/:id/incr', (req, res) => {
  console.log('actor', req.params.id, 'incr', req.body)
  res.json(req.body + 1)
})

app.post('/actor/foo/:id/echo', (req, res) => {
  if (req.body && req.body.msg) {
    console.log(`actor ${req.params.id} says "${req.body.msg}"`)
  } else {
    console.log(`actor ${req.params.id} has nothing to say`)
  }
  res.json('OK')
})

app.use(errorHandler) // enable kar error handling

const server = app.listen(process.env.KAR_APP_PORT, '127.0.0.1')
