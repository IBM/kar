const express = require('express')
const { actor, sys } = require('kar')
const { v4: uuidv4 } = require('uuid')

const verbose = process.env.VERBOSE

class Fork {
  async activate () {
    this.inUseBy = await actor.state.get(this, 'inUseBy') || 'nobody'
  }

  async pickUp (who) {
    if (this.inUseBy === 'nobody') {
      this.inUseBy = who
      await actor.state.set(this, 'inUseBy', who)
      return true
    } else if (this.inUseBy === who) {
      // can happen if pickUp is re-executed due to a failure
      return true
    } else {
      return false
    }
  }

  async putDown (who) {
    if (this.inUseBy === who) {
      // not guaranteed if putDown is re-executed due to failure
      this.inUseBy = 'nobody'
      await actor.state.set(this, 'inUseBy', this.inUseBy)
    }
  }
}

class Philosopher {
  async activate () {
    Object.assign(this, await actor.state.getAll(this))
  }

  nextStepTime () {
    const thinkTime = Math.floor(Math.random() * 1000) // random 0...999ms
    return new Date(Date.now() + thinkTime)
  }

  async joinTable (table, firstFork, secondFork, targetServings, step) {
    if (this.step !== step) throw new Error('unexpected step')
    step = uuidv4()
    this.table = table
    this.firstFork = firstFork
    this.secondFork = secondFork
    this.servingsEaten = 0
    this.targetServings = targetServings
    await actor.reminders.schedule(this, 'getFirstFork', { id: 'step', targetTime: this.nextStepTime() }, 1, step)
    this.step = step
    await actor.state.setMultiple(this, { step, table, firstFork, secondFork, servingsEaten: this.servingsEaten, targetServings })
  }

  async getFirstFork (attempt, step) {
    if (this.step !== step) throw new Error('unexpected step')
    step = uuidv4()
    if (await actor.call(actor.proxy('Fork', this.firstFork), 'pickUp', this.kar.id)) {
      await actor.tell(this, 'getSecondFork', 1, step)
    } else {
      if (attempt > 5) {
        console.log(`Warning: Philosopher ${this.kar.id} has failed to acquire his first Fork ${attempt} times`)
      }
      await actor.reminders.schedule(this, 'getFirstFork', { id: 'step', targetTime: this.nextStepTime() }, attempt + 1, step)
    }
    this.step = step
    await actor.state.set(this, 'step', step)
  }

  async getSecondFork (attempt, step) {
    if (this.step !== step) throw new Error('unexpected step')
    step = uuidv4()
    if (await actor.call(actor.proxy('Fork', this.secondFork), 'pickUp', this.kar.id)) {
      await actor.tell(this, 'eat', step)
    } else {
      if (attempt > 5) {
        console.log(`Warning: Philosopher ${this.kar.id} has failed to acquire his second Fork ${attempt} times`)
      }
      await actor.reminders.schedule(this, 'getSecondFork', { id: 'step', targetTime: this.nextStepTime() }, attempt + 1, step)
    }
    this.step = step
    await actor.state.set(this, 'step', step)
  }

  async eat (step) {
    if (this.step !== step) throw new Error('unexpected step')
    step = uuidv4()
    if (verbose) console.log(`${this.kar.id} ate serving number ${this.servingsEaten}`)
    await actor.call(actor.proxy('Fork', this.secondFork), 'putDown', this.kar.id)
    await actor.call(actor.proxy('Fork', this.firstFork), 'putDown', this.kar.id)
    if (this.servingsEaten < this.targetServings) {
      await actor.reminders.schedule(this, 'getFirstFork', { id: 'step', targetTime: this.nextStepTime() }, 1, step)
    } else {
      await actor.call(actor.proxy('Table', this.table), 'doneEating', this.kar.id)
    }
    this.servingsEaten = this.servingsEaten + 1
    this.step = step
    await actor.state.setMultiple(this, { step, servingsEaten: this.servingsEaten })
  }
}

class Table {
  async activate () {
    const that = await actor.state.getAll(this)
    this.cafe = that.cafe
    this.diners = new Set(that.diners || [])
    this.step = that.step
  }

  occupancy () {
    return this.diners.size
  }

  async set (cafe, n, servings, step) {
    if (this.step !== step) throw new Error('unexpected step')
    step = uuidv4()
    this.cafe = cafe
    console.log(`Cafe ${cafe} is seating table ${this.kar.id} with ${n} hungry philosophers for ${servings} servings`)
    var philosophers = []
    for (var i = 0; i < n; i++) {
      philosophers[i] = `${cafe}-${this.kar.id}-${i}`
      this.diners.add(philosophers[i])
    }
    await actor.tell(this, 'eat', n, servings, step)
    this.step = step
    await actor.state.setMultiple(this, { step, diners: Array.from(this.diners) })
  }

  async eat (n, servings, step) {
    if (this.step !== step) throw new Error('unexpected step')
    step = uuidv4()
    var philosophers = []
    var forks = []
    for (var i = 0; i < n; i++) {
      philosophers[i] = `${this.cafe}-${this.kar.id}-${i}`
      forks[i] = `${this.cafe}-${this.kar.id}-${i}`
    }
    for (i = 0; i < n - 1; i++) {
      await actor.call(actor.proxy('Philosopher', philosophers[i]), 'joinTable', this.kar.id, forks[i], forks[i + 1], servings)
    }
    await actor.call(actor.proxy('Philosopher', philosophers[n - 1]), 'joinTable', this.kar.id, forks[0], forks[n - 1], servings)
    this.step = step
    await actor.state.set(this, 'step', step)
  }

  async doneEating (philosopher) {
    this.diners.delete(philosopher)
    await actor.state.set(this, 'diners', Array.from(this.diners))
    console.log(`Philosopher ${philosopher} is done eating; there are now ${this.diners.size} present at the table`)
    if (this.diners.size === 0) {
      console.log(`Table ${this.kar.id} is now empty!`)
    }
  }
}

class Cafe {
  async occupancy (table) {
    return actor.call(actor.proxy('Table', table), 'occupancy')
  }

  async seatTable (n = 5, servings = 20, table = uuidv4()) {
    await actor.call(actor.proxy('Table', table), 'set', this.kar.id, n, servings)
    return table
  }
}

// Server setup: register actors with KAR and start express
const app = express()
app.use(sys.actorRuntime({ Fork, Philosopher, Table, Cafe }))
app.listen(process.env.KAR_APP_PORT, '127.0.0.1')
