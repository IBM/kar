/*
 * Copyright IBM Corporation 2020,2021
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
    if (this.inUseBy === who) { // can be false if putDown is re-executed due to failure
      this.inUseBy = 'nobody'
      await actor.state.set(this, 'inUseBy', this.inUseBy)
    }
  }
}

class Philosopher {
  async activate () {
    Object.assign(this, await actor.state.getAll(this))
    this.step = this.step || this.kar.id // use actor id as initial step
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
    this.n = that.n
    this.diners = that.diners || [] // initial state is an empty table
    this.step = that.step || this.kar.id // use actor id as initial step
  }

  occupancy () { return this.diners.length }

  philosopher (p) { return `${this.cafe}-${this.kar.id}-philosopher-${p}` }

  fork (f) { return `${this.cafe}-${this.kar.id}-fork-${f}` }

  async prepare (cafe, n, servings, step) {
    if (this.step !== step) throw new Error('unexpected step')
    step = uuidv4()
    this.cafe = cafe
    this.n = n
    for (var i = 0; i < n; i++) {
      this.diners[i] = this.philosopher(i)
    }
    console.log(`Cafe ${cafe} is seating table ${this.kar.id} with ${n} hungry philosophers for ${servings} servings`)
    await actor.tell(this, 'serve', servings, step)
    this.step = step
    await actor.state.setMultiple(this, { step, cafe, n, diners: this.diners })
  }

  async serve (servings, step) {
    if (this.step !== step) throw new Error('unexpected step')
    step = uuidv4()
    for (var i = 0; i < this.n - 1; i++) {
      const who = this.philosopher(i)
      const fork1 = this.fork(i)
      const fork2 = this.fork(i + 1)
      await actor.call(actor.proxy('Philosopher', who), 'joinTable', this.kar.id, fork1, fork2, servings, who)
    }
    const who = this.philosopher(this.n - 1)
    const fork1 = this.fork(0)
    const fork2 = this.fork(this.n - 1)
    await actor.call(actor.proxy('Philosopher', who), 'joinTable', this.kar.id, fork1, fork2, servings, who)
    this.step = step
    await actor.state.set(this, 'step', step)
  }

  async doneEating (philosopher) {
    this.diners = this.diners.filter(x => x !== philosopher)
    await actor.state.set(this, 'diners', this.diners)
    console.log(`Philosopher ${philosopher} is done eating; there are now ${this.diners.length} present at the table`)
    if (this.diners.length === 0) {
      console.log(`Table ${this.kar.id} is now empty!`)
      const step = uuidv4()
      await actor.tell(this, 'busTable', step)
      this.step = step
      await actor.state.set(this, 'step', step)
    }
  }

  async busTable (step) {
    if (this.step !== step) throw new Error('unexpected step')
    step = uuidv4()
    for (var i = 0; i < this.n; i++) {
      await actor.remove(actor.proxy('Philosopher', this.philosopher(i)))
      await actor.remove(actor.proxy('Fork', this.fork(i)))
    }
    await actor.remove(this)
    this.step = step
    await actor.state.set(this, 'step', step)
  }
}

class Cafe {
  async occupancy (table) {
    return actor.call(actor.proxy('Table', table), 'occupancy')
  }

  async seatTable (n = 5, servings = 20, requestId = uuidv4()) {
    await actor.call(actor.proxy('Table', requestId), 'prepare', this.kar.id, n, servings, requestId)
    return requestId
  }
}

// Server setup: register actors with KAR and start express
const app = express()
app.use(sys.actorRuntime({ Fork, Philosopher, Table, Cafe }))
app.listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')
