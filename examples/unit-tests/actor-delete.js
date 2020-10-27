const { actor, sys } = require('kar')
const express = require('express')
const app = express()

class Victim {
  constructor (id) {
    this.id = id
  }

  async activate () {
    console.log('actor', this.id, 'activate')
    const state = await actor.state.getAll(this)
    this.timestamp = state.timestamp || Date.now()
  }

  async deactivate () {
    console.log('actor', this.id, 'deactivate')
    await this.checkpoint()
  }

  getStamp () {
    return this.timestamp
  }

  async checkpoint () {
    console.log(`checkpoint invoked ${this.timestamp}`)
    await actor.state.set(this, 'timestamp', this.timestamp)
  }

  async deleteMyself () {
    await actor.purge(this)
  }

  async deleteMyState () {
    await actor.state.removeAll(this)
  }

  async deleteOther (id) {
    await actor.purge(actor.proxy('Victim', id))
  }
}

app.use(sys.actorRuntime({ Victim }))
app.listen(process.env.KAR_APP_PORT, '127.0.0.1')
