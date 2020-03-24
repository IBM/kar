const express = require('express')
const { logger, jsonParser, errorHandler } = require('kar')

const app = express()

app.use(logger, jsonParser)

app.post('/incr', (req, res) => {
  console.log('incr', req.body)
  setTimeout(() => res.json(req.body + 1), 10000)
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
  setTimeout(() => res.json(req.body + 1), 10000)
})

app.use(errorHandler)

app.listen(process.env.KAR_APP_PORT, '127.0.0.1')
