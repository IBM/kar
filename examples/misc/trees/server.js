const express = require('express')
const { actor, sys } = require('kar')

// Sequential tree construction
class Sync {
  async test (depth) {
    this.count = Math.pow(2, depth - 1)
    console.log('expecting', this.count, 'leaves')
    this.startTime = Date.now()
    await actor.call(this, this, 'fork', depth)
    console.log('sync test duration:', Date.now() - this.startTime)
  }

  async fork (depth) {
    if (--depth > 0) {
      await actor.call(this, actor.proxy('Sync', this.kar.id * 2), 'fork', depth)
      await actor.call(this, actor.proxy('Sync', this.kar.id * 2 + 1), 'fork', depth)
    }
  }
}

// Parallel tree construction
// Wait for all leaves to report back to the root
class Async {
  async test (depth) {
    this.count = Math.pow(2, depth - 1)
    console.log('expecting', this.count, 'leaves')
    this.startTime = Date.now()
    const promise = new Promise(resolve => { this.resolve = resolve })
    actor.call(this, this, 'fork', depth, this.kar.session)
    await promise
    console.log('async test duration:', Date.now() - this.startTime)
  }

  decr () {
    // node guarantees there is no race even KAR makes concurrent invocations
    if (--this.count <= 0) {
      this.resolve()
    }
  }

  fork (depth, session) {
    if (--depth === 0) {
      // force session ID to permit the invocation of decr while test is still running
      actor.call({ kar: { session } }, actor.proxy('Async', 1), 'decr')
    } else {
      actor.tell(actor.proxy('Async', this.kar.id * 2), 'fork', depth, session)
      actor.tell(actor.proxy('Async', this.kar.id * 2 + 1), 'fork', depth, session)
    }
  }
}

// Parallel tree construction
// Wait for left and right subtrees construction to complete at each level
// Requires HTTP/2 to handle the many concurrent HTTP connections
class Par {
  async test (depth) {
    this.count = Math.pow(2, depth - 1)
    console.log('expecting', this.count, 'leaves')
    this.startTime = Date.now()
    await actor.call(this, this, 'fork', depth)
    console.log('parallel test duration:', Date.now() - this.startTime)
  }

  async fork (depth) {
    if (--depth > 0) {
      const future = await actor.asyncCall(this, actor.proxy('Par', this.kar.id * 2), 'fork', depth)
      await actor.call(this, actor.proxy('Par', this.kar.id * 2 + 1), 'fork', depth)
      await future()
    }
  }
}

const app = express()
app.use(sys.actorRuntime({ Sync, Async, Par }))
sys.h2c(app).listen(process.env.KAR_APP_PORT, '127.0.0.1')
