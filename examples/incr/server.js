const express = require('express')
const { logger, preprocessor, postprocessor, shutdown } = require('./kar')

const app = express()

app.use(logger, preprocessor)

app.post('/incr', (req, res) => {
  console.log('incr', req.body)
  res.json(req.body + 1)
})

async function doShutdown () {
  await shutdown()
  server.close(() => process.exit())
}
app.post('/shutdown', (reg, res) => {
  console.log('Shutting down service')
  doShutdown()
})

app.use(postprocessor)

const server = app.listen(process.env.KAR_APP_PORT, '127.0.0.1')
