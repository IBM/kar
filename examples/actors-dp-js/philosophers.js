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
const { actor, sys } = require('kar-sdk')
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
  }

  think () {
    return new Promise(resolve => setTimeout(resolve, Math.floor(Math.random() * 1000))) // random 0...999ms
  }

  async joinTable (table, firstFork, secondFork, targetServings) {
    this.table = table
    this.firstFork = firstFork
    this.secondFork = secondFork
    this.servingsEaten = 0
    this.targetServings = targetServings
    await actor.state.setMultiple(this, { table, firstFork, secondFork, servingsEaten: this.servingsEaten, targetServings })
    await this.think()
    return actor.tailCall(this, 'getFirstFork', 1)
  }

  async getFirstFork (attempt) {
    if (await actor.call(actor.proxy('Fork', this.firstFork), 'pickUp', this.kar.id)) {
      return actor.tailCall(this, 'getSecondFork', 1)
    } else {
      if (attempt > 5) {
        console.log(`Warning: Philosopher ${this.kar.id} has failed to acquire his first Fork ${attempt} times`)
      }
      await this.think()
      return actor.tailCall(this, 'getFirstFork', attempt + 1)
    }
  }

  async getSecondFork (attempt) {
    if (await actor.call(actor.proxy('Fork', this.secondFork), 'pickUp', this.kar.id)) {
      return actor.tailCall(this, 'eat', this.servingsEaten)
    } else {
      if (attempt > 5) {
        console.log(`Warning: Philosopher ${this.kar.id} has failed to acquire his second Fork ${attempt} times`)
      }
      await this.think()
      return actor.tailCall(this, 'getSecondFork', attempt + 1)
    }
  }

  async eat (serving) {
    if (verbose) console.log(`${this.kar.id} ate serving number ${serving}`)
    await actor.call(actor.proxy('Fork', this.secondFork), 'putDown', this.kar.id)
    await actor.call(actor.proxy('Fork', this.firstFork), 'putDown', this.kar.id)
    this.servingsEaten = serving + 1
    await actor.state.set(this, 'servingsEaten', this.servingsEaten)
    if (this.servingsEaten < this.targetServings) {
      await this.think()
      return actor.tailCall(this, 'getFirstFork', 1)
    } else {
      return actor.tailCall(actor.proxy('Table', this.table), 'doneEating', this.kar.id)
    }
  }
}

class Table {
  async activate () {
    const that = await actor.state.getAll(this)
    this.cafe = that.cafe
    this.n = that.n
    this.diners = that.diners || [] // initial state is an empty table
  }

  occupancy () { return this.diners.length }

  philosopher (p) { return `${this.cafe}-${this.kar.id}-philosopher-${p}` }

  fork (f) { return `${this.cafe}-${this.kar.id}-fork-${f}` }

  async prepare (cafe, n, servings) {
    this.cafe = cafe
    this.n = n
    for (var i = 0; i < n; i++) {
      this.diners[i] = this.philosopher(i)
    }
    await actor.state.setMultiple(this, { cafe, n, diners: this.diners })
    console.log(`Cafe ${cafe} has seated table ${this.kar.id} with ${n} hungry philosophers for ${servings} servings`)
    return actor.tailCall(this, 'serve', servings)
  }

  async serve (servings) {
    for (var i = 0; i < this.n - 1; i++) {
      const who = this.philosopher(i)
      const fork1 = this.fork(i)
      const fork2 = this.fork(i + 1)
      await actor.tell(actor.proxy('Philosopher', who), 'joinTable', this.kar.id, fork1, fork2, servings)
    }
    const who = this.philosopher(this.n - 1)
    const fork1 = this.fork(0)
    const fork2 = this.fork(this.n - 1)
    await actor.tell(actor.proxy('Philosopher', who), 'joinTable', this.kar.id, fork1, fork2, servings)
  }

  async doneEating (philosopher) {
    this.diners = this.diners.filter(x => x !== philosopher)
    await actor.state.set(this, 'diners', this.diners)
    console.log(`Philosopher ${philosopher} is done eating; there are now ${this.diners.length} present at the table`)
    if (this.diners.length === 0) {
      return actor.tailCall(this, 'busTable')
    }
  }

  async busTable () {
    console.log(`Table ${this.kar.id} is now empty!`)
    for (var i = 0; i < this.n; i++) {
      await actor.remove(actor.proxy('Philosopher', this.philosopher(i)))
      await actor.remove(actor.proxy('Fork', this.fork(i)))
    }
    await actor.remove(this)
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
