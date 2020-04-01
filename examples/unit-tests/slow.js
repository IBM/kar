const express = require('express')
const { logger, jsonParser, errorHandler } = require('kar')

const app = express()

app.use(logger, jsonParser)

app.post('/incr', (req, res) => {
  console.log('incr', req.body)
  setTimeout(() => res.json(req.body + 1), 10000)
})

app.use(errorHandler)

app.listen(process.env.KAR_APP_PORT, '127.0.0.1')
