const express = require('express')
const { actor, sys } = require('kar')

const app = express()

class Test {
  async A () {
    console.log('entering method A')
    await actor.call(this, this, 'B') // synchronous call to self within the same session -> OK
    console.log('exiting method A')
  }

  async B () {
    console.log('entering method B')
    await actor.call(this, 'A') // synchronous call to self in a new session -> deadlock
    console.log('exiting method B')
  }
}

app.use(sys.actorRuntime({ Test }))

app.listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')

async function main () {
  try {
    await actor.call(actor.proxy('Test', '123'), 'A')
  } catch (err) {
    console.error(err)
  }
}

main()
