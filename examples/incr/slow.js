const express = require('express')
const { logger, preprocessor, postprocessor } = require('kar')

const app = express()

app.use(logger, preprocessor)

app.post('/incr', (req, res) => {
  console.log('incr', req.body)
  setTimeout(() => res.json(req.body + 1), 10000)
})

app.use(postprocessor)

app.listen(process.env.KAR_APP_PORT, '127.0.0.1')
