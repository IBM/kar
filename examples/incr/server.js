const express = require('express')
const { logger, preprocessor, postprocessor, shutdown } = require('./kar')

async function doShutdown (reg, res) {
  console.log('Shutting down service')
  await shutdown()
  await server.close(() => process.exit())
}

const app = express()

app.use(logger, preprocessor)

app.post('/incr', (req, res) => {
  console.log('incr', req.body)
  res.json(req.body + 1)
})

app.post('/shutdown', doShutdown)

app.use(postprocessor)

const server = app.listen(process.env.KAR_APP_PORT, '127.0.0.1')
