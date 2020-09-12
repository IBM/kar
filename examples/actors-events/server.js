const express = require('express')
const { actor, events, sys } = require('kar')

const app = express()

class Handler {
  handler (event) {
    console.log(event)
  }
}

app.use(sys.actorRuntime({ Handler }))

app.listen(process.env.KAR_APP_PORT, '127.0.0.1')

// main function
async function main () {
  console.log('subscribe:', await events.subscribe(actor.proxy('Handler', 'test-actor'), 'handler', 'test-topic'))
}

main()
