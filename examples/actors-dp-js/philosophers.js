const express = require('express')
const { actor, sys } = require('kar')
const { v4: uuidv4 } = require('uuid')

class Fork {
  async activate () {
    this.inUseBy = await actor.state.get(this, 'inUseBy') || 'nobody'
  }

  async deactivate () {
    await actor.state.set(this, 'inUseBy', this.inUseBy)
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
      this.inUseBy = 'nobody'
      await actor.state.set(this, 'inUseBy', this.inUseBy)
      return true
    } else {
      // can happen when putDown is re-executed due to a failure
      return false
    }
  }
}

class Philosopher {
  async activate () {
    Object.assign(this, await actor.state.getAll(this))
  }

  async deactivate () {
    await this.checkpointState()
  }

  async checkpointState () {
    const state = {
      cafe: this.cafe,
      firstFork: this.firstFork,
      secondFork: this.secondFork,
      servingsEaten: this.servingsEaten,
      targetServings: this.targetServings
    }
    await actor.state.setMultiple(this, state)
  }

  nextStepTime () {
    const thinkTime = Math.floor(Math.random() * 1000) // random 0...999ms
    return new Date(Date.now() + thinkTime)
  }

  async joinTable (cafe, firstFork, secondFork, targetServings) {
    this.cafe = cafe
    this.firstFork = firstFork
    this.secondFork = secondFork
    this.servingsEaten = 0
    this.targetServings = targetServings
    this.checkpointState()
    await actor.reminders.schedule(this, 'getFirstFork', { id: 'step', targetTime: this.nextStepTime() }, 1)
  }

  async getFirstFork (attempt) {
    if (await actor.call(actor.proxy('Fork', this.firstFork), 'pickUp', this.kar.id)) {
      await actor.tell(this, 'getSecondFork', 1)
    } else {
      if (attempt > 5) {
        console.log(`Warning: ${this.kar.id} has failed to acquire his first Fork ${attempt} times`)
      }
      await actor.reminders.schedule(this, 'getFirstFork', { id: 'step', targetTime: this.nextStepTime() }, attempt + 1)
    }
  }

  async getSecondFork (attempt) {
    if (await actor.call(actor.proxy('Fork', this.secondFork), 'pickUp', this.kar.id)) {
      await actor.tell(this, 'eat', this.servingsEaten)
    } else {
      if (attempt > 5) {
        console.log(`Warning: ${this.kar.id} has failed to acquire his second Fork ${attempt} times`)
      }
      await actor.reminders.schedule(this, 'getSecondFork', { id: 'step', targetTime: this.nextStepTime() }, attempt + 1)
    }
  }

  async eat (servingsEaten) {
    console.log(`${this.kar.id} ate serving number ${servingsEaten}`)
    this.servingsEaten = servingsEaten + 1
    await this.checkpointState()
    await actor.call(actor.proxy('Fork', this.secondFork), 'putDown', this.kar.id)
    await actor.call(actor.proxy('Fork', this.firstFork), 'putDown', this.kar.id)
    if (this.servingsEaten < this.targetServings) {
      await actor.reminders.schedule(this, 'getFirstFork', { id: 'step', targetTime: this.nextStepTime() }, 1)
    } else {
      await actor.call(actor.proxy('Cafe', this.cafe), 'doneEating', this.kar.id)
    }
  }
}

class Cafe {
  async activate () {
    const da = await actor.state.get(this, 'diners') || []
    this.diners = new Set(da)
  }

  async deactivate () {
    await actor.state.set(this, 'diners', Array.from(this.diners))
  }

  async seatTable (n = 5, servings = 20) {
    console.log(`Cafe ${this.kar.id} is seating a new table of ${n} hungry philosophers for ${servings} servings`)
    var philosophers = []
    var forks = []
    for (var i = 0; i < n; i++) {
      philosophers[i] = uuidv4()
      this.diners.add(philosophers[i])
      forks[i] = uuidv4()
    }
    for (i = 0; i < n - 1; i++) {
      await actor.call(actor.proxy('Philosopher', philosophers[i]), 'joinTable', this.kar.id, forks[i], forks[i + 1], servings)
    }
    await actor.call(actor.proxy('Philosopher', philosophers[n - 1]), 'joinTable', this.kar.id, forks[0], forks[n - 1], servings)
    await actor.state.set(this, 'diners', Array.from(this.diners))
  }

  async doneEating (philosopher) {
    this.diners.delete(philosopher)
    await actor.state.set(this, 'diners', Array.from(this.diners))
    console.log(`Cafe ${this.kar.id}: ${philosopher} is done eating; there are now ${this.diners.size} present`)
    if (this.diners.size === 0) {
      console.log(`Cafe ${this.kar.id} is now empty!`)
    }
  }
}

// Server setup: register actors with KAR and start express
const app = express()
app.use(sys.actorRuntime({ Fork, Philosopher, Cafe }))
app.listen(process.env.KAR_APP_PORT, '127.0.0.1')
