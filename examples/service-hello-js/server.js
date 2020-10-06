const express = require('express')

if (!process.env.KAR_APP_PORT) {
  console.error('KAR_APP_PORT must be set. Aborting.')
  process.exit(1)
}

// create an express application
const app = express()

// parse request bodies with text/plain content type to json
app.use(express.text())

// parse request bodies with application/json content type to json
app.use(express.json())

// a route that accepts and returns text/plain
app.post('/helloText', (req, res) => {
  const msg = `Hello ${req.body}!`
  console.log(msg)
  res.send(msg)
})

// a route that accepts and returns application/json
app.post('/helloJson', (req, res) => {
  const msg = `Hello ${req.body.name}!`
  console.log(msg)
  res.json({ greetings: msg })
})

// start server on port $KAR_APP_PORT
console.log('Starting greetings server...')
app.listen(process.env.KAR_APP_PORT, '127.0.0.1')
