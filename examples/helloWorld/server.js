const express = require('express')
const { sys } = require('kar')

const app = express()

app.use(sys.logger, sys.jsonParser) // enable kar logging and parsing
app.use(sys.errorHandler) // enable kar error handling

// Define the route this service provides
app.post('/hello', (req, res) => {
  const msg = `Hello ${req.body}!`
  console.log(msg)
  res.json(msg)
})

app.listen(process.env.KAR_APP_PORT, '127.0.0.1')
