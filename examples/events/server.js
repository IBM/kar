const express = require('express')
const fetch = require('fetch-retry')(require('node-fetch'), { retries: 10 })

const base = `http://127.0.0.1:${process.env.KAR_RUNTIME_PORT}/kar/v1`

// subscribe a route to a topic
async function subscribe (topic, path) {
  const res = await fetch(`${base}/event/${topic}/subscribe`, { method: 'POST', body: JSON.stringify({ path }) })
  return res.text()
}

// create an express application
const app = express()

// parse events
app.use(express.json({ type: 'application/cloudevents+json' }))

// event handler
app.post('/handle', async (req, res) => {
  console.log('event received:', req.body)
  res.send('OK')
})

// start server on port $KAR_APP_PORT
app.listen(process.env.KAR_APP_PORT, '127.0.0.1')

async function main () {
  // subscribe to topic
  console.log('subscribe:', await subscribe('test-topic', '/handle'))
}

main()
