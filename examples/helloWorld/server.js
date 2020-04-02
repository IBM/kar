const express = require('express')
const { logger, jsonParser, errorHandler } = require('kar')

const app = express()

app.use(logger, jsonParser) // enable kar logging and parsing
app.use(errorHandler) // enable kar error handling

// Define the route this service provides
app.post('/hello', (req, res) => {
  const msg = `Hello ${req.body}!`
  console.log(msg)
  res.json(msg)
})

app.listen(process.env.KAR_APP_PORT, '127.0.0.1')
