/*
 * Copyright IBM Corporation 2020,2022
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

const { actor, events, sys } = require('kar-sdk')

// CloudEvents SDK for defining a structured HTTP request receiver.
const { CloudEvent } = require('cloudevents')

const truthy = s => s && s.toLowerCase() !== 'false' && s !== '0'
const verbose = truthy(process.env.VERBOSE)
const debug = truthy(process.env.DEBUG)

function randI (max) { return Math.floor(Math.random() * Math.floor(max)) }

// The states a Researcher can be in
const States = {
  ONBOARDING: 'onboarding',
  HOME: 'home',
  COMMUTING: 'commuting',
  WORKING: 'working',
  MEETING: 'meeting',
  COFFEE: 'coffee',
  LUNCH: 'lunch'
}

const bucketSizeInMS = 100

class Company {
  get name () { return this.kar.id }

  async checkpoint () {
    const state = {
      nextSerialNumber: this.nextSerialNumber,
      sites: this.sites,
      bluepages: this.bluepages
    }
    await actor.state.setMultiple(this, state)
  }

  async activate () {
    const state = await actor.state.getAll(this)
    this.nextSerialNumber = state.nextSerialNumber || 0
    this.sites = state.sites || []
    this.bluepages = state.bluepages || {}

    await events.createTopic('siteReport')
    await events.createTopic('outputReport')
  }

  async deactivate () {
    await this.checkpoint()
  }

  get count () { return Object.keys(this.bluepages).length }

  activeEmployees (site) {
    const ans = []
    for (const sn in this.bluepages) {
      if (this.bluepages[sn] === site) {
        ans.push(sn)
      }
    }
    return ans
  }

  async hire ({ site = 'Yorktown', workers = 3, days = 1, steps = 20, thinkms = 10000 } = {}) {
    if (!this.sites.includes(site)) {
      this.sites.push(site)
      await actor.call(this, actor.proxy('Site', site), 'joinCompany', this.name)
      await actor.reminders.schedule(actor.proxy('Site', site), 'siteReport',
        { id: 'aisle14', targetTime: new Date(Date.now() + 1000) }, '5s')
      await actor.state.set(this, 'sites', this.sites)
    }

    console.log(`${workers} hired to perform ${steps} tasks/day for ${days} days at ${site}`)

    const sn = this.nextSerialNumber
    this.nextSerialNumber = sn + workers
    await actor.state.set(this, 'nextSerialNumber', this.nextSerialNumber)

    for (var i = 0; i < workers; i++) {
      const name = sn + i
      this.bluepages[name] = site
      await actor.tell(actor.proxy('Site', site), 'newHire', name, days, steps, thinkms)
    }
    await actor.state.set(this, 'bluepages', this.bluepages)
  }

  async retire (who) {
    delete this.bluepages[who]
    await actor.state.set(this, 'bluepages', this.bluepages)
  }
}

// Sites update rapidly changing aggregate statistics by processing
// the workerUpdate and retire messages from individual workers.
//
// These statistics are periodically reported via siteReport.
//
// Rather than checkpointing this data on every change, we will instead
// detect in `activate` when a Site is being loaded from a potentially
// outdated checkpoint and take corrective actions by rebuilding the list
// of employees currently assigned to the site and relying on workerUpdate
// future workerUpdate messages to gradually recover a view on site activity.
class Site {
  get name () { return this.kar.id }
  get count () { return Object.keys(this.workers).length }

  async checkpoint () {
    const state = {
      reminderDelays: this.reminderDelays,
      workerUpdateLatency: this.workerUpdateLatency,
      workers: this.workers
    }
    await actor.state.setMultiple(this, state)
  }

  async activate () {
    const state = await actor.state.getAll(this)
    this.reminderDelays = state.reminderDelays || []
    this.workerUpdateLatency = state.workerUpdateLatency || []
    this.workers = state.workers || {}
    this.company = state.company
    await actor.state.set(this, 'cleanShutdown', false)
    if (this.company !== undefined && state.cleanShutdown !== true) {
      const employees = await actor.call(actor.proxy('Company', this.company), 'activeEmployees', this.name)
      for (const sn of employees) {
        this.workers[sn] = States.ONBOARDING // Not accurate, but will be corrected on next workerUpdate
      }
    }
    if (debug) console.log(`activated Site ${this.name} with occupants ${state.workers}`)
  }

  async deactivate () {
    await this.checkpoint()
    await actor.state.set(this, 'cleanShutdown', true)
    if (debug) console.log(`deactivated Site ${this.name}`)
  }

  async joinCompany (company) {
    this.company = company
    await actor.state.set(this, 'company', company)
  }

  async newHire (who, days, steps, thinkms) {
    this.workers[who] = States.ONBOARDING
    await actor.state.set(this, 'workers', this.workers)
    await actor.tell(actor.proxy('Researcher', who), 'newHire', this.name, days, steps, thinkms)
  }

  async retire (who, delays = []) {
    if (this.workers[who]) {
      const ds = this.reminderDelays
      delays.forEach(function (missedMS, _) {
        const missedBucket = Math.floor(missedMS / bucketSizeInMS)
        ds[missedBucket] = (ds[missedBucket] || 0) + 1
      })
      delete this.workers[who]
      await actor.state.setMultiple(this, { workers: this.workers, reminderDelays: this.reminderDelays })
    }

    await actor.call(this, actor.proxy('Company', this.company), 'retire', who)
    if (verbose) console.log(`Researcher ${who} has retired. Site employee count is now ${this.count}`)
  }

  async workerUpdate (who, activity, timestamp) {
    if (this.workers[who] !== undefined) {
      this.workers[who] = activity
      const lag = Date.now() - timestamp
      const missedBucket = Math.floor(lag / bucketSizeInMS)
      this.workerUpdateLatency[missedBucket] = (this.workerUpdateLatency[missedBucket] || 0) + 1
    }
  }

  async siteReport () {
    const siteEmployees = this.count
    const time = new Date().toString()
    const status = { site: this.name, siteEmployees, time }
    status.bucketSizeInMS = bucketSizeInMS
    status.reminderDelays = this.reminderDelays
    status.workerUpdateLatency = this.workerUpdateLatency
    if (siteEmployees > 0) {
      for (const s in States) {
        status[States[s]] = 0
      }
      for (const worker in this.workers) {
        status[this.workers[worker]] += 1
      }
      console.log(status)
    }

    // Construct Cloud Event containing the status report.
    var reportEvent = new CloudEvent({
      type: 'site.report',
      source: 'javascript.client',
      data: status
    })

    if (verbose) console.log(`Publish event: ${reportEvent}`)

    // Publish report as an event.
    events.publish('siteReport', reportEvent)

    return status
  }

  async resetDelayStats () {
    this.reminderDelays = []
    this.workerUpdateLatency = []
    await actor.state.setMultiple(this, { reminderDelays: [], workerUpdateLatency: [] })
  }
}

// An Office's Actor ID is of the form Site:Office
//
// Offices are an example of an actor with non-essential state
// that is intentionally not preserved across failures.
// State is only checkpointed in deactivate and the enter/leave
// operations are implemented to ignore non-sensical updates
// (eg a Reseeacher leaving an office that are not in)
class Office {
  static coffeeShop (site) { return `${site}:OutTakes` }
  static cafeteria (site) { return `${site}:Cafeteria` }
  static randomOffice (site) {
    // TODO: Introduce accurate site-specific office numbering patterns just for fun.
    const floor = randI(3)
    const aisle = 1 + randI(40)
    const office = 1 + randI(64)
    return `${site}:${aisle}-${floor}${office}`
  }

  get name () { return this.kar.id }

  isEmpty () { return this.occupants.size === 0 }
  get count () { return this.occupants.size }

  async activate () {
    const so = await actor.state.get(this, 'occupants') || []
    this.occupants = new Set(so)
    if (debug) console.log(`activated Office ${this.name} with occupancy ${this.count}`)
  }

  async deactivate () {
    await actor.state.set(this, 'occupants', Array.from(this.occupants))
    if (debug) console.log(`deactivated Office ${this.name}`)
  }

  async enter (who) {
    this.occupants.add(who)
    if (debug) console.log(`${who} entered Office ${this.name} occupancy is now ${this.count}`)
  }

  async leave (who) {
    this.occupants.delete(who)
    if (debug) console.log(`${who} left Office ${this.name} occupancy is now ${this.count}`)
  }
}

// Researchers are the active entities in this simulation.
//
// The class illustrates how to optimize actor checkpointing
// operation by separating the management of initialize-only state
// from other properties of the object.
//
// The `move` method is triggered by a one-shot reminder
// (time triggered event).  It illustrates a general pattern
// fault-tolerance pattern of breaking a complex operation
// into a sequence of steps.
//
class Researcher {
  get name () { return this.kar.id }

  async newHire (site, days, steps, thinkms) {
    const initialState = { site, career: days * steps, workday: steps, thinkms, activity: States.ONBOARDING, delays: [], currentStep: 0 }
    Object.assign(this, initialState)
    await actor.state.setMultiple(this, initialState)

    const when = new Date(Date.now() + thinkms)
    await actor.reminders.schedule(this, 'move', { id: 'step', targetTime: when }, when.getTime())
  }

  // Checkpoint only saves the transitory state of the Researcher
  // All initialize-only fields are persisted in newHire.
  async checkpointState () {
    const state = {
      activity: this.activity,
      currentStep: this.currentStep,
      location: this.location,
      delays: this.delays
    }
    await actor.state.setMultiple(this, state)
  }

  async activate () {
    Object.assign(this, await actor.state.getAll(this))
    if (debug) console.log(`activated Researcher ${this.name} with state `, this)
  }

  async deactivate () {
    await this.checkpointState()
    if (debug) console.log(`deactivated Researcher ${this.name}`)
  }

  async move (targetTime) {
    const observedDelay = Date.now() - targetTime
    if (debug) console.log(`${this.site}: Researcher ${this.name} started move ${this.currentStep} with delay ${observedDelay}`)

    if (this.location !== undefined) {
      await actor.call(this, actor.proxy('Office', this.location), 'leave', this.name)
    }

    this.delays[this.currentStep] = observedDelay

    // TODO: atomic checkpoint & doNext
    await this.checkpointState()
    if (this.currentStep === this.career) {
      await actor.tell(actor.proxy('Site', this.site), 'retire', this.name, this.delays)
      await actor.remove(this)
    } else {
      await actor.tell(this, 'determineNextStep')
    }
  }

  // Commit to the next action; invoked as continuation to move
  async determineNextStep () {
    const priorActivity = this.activity
    const diceRoll = Math.random()
    let thinkTime = 1 + randI(this.thinkms)
    if (this.currentStep % this.workday === 0) {
      // Morning rush hour.
      this.activity = States.COMMUTING
      thinkTime = thinkTime * 3
    } else if ((this.currentStep + 2) % this.workday === 0) {
      // Evening rush hour.
      this.activity = States.COMMUTING
      thinkTime = thinkTime * 2
    } else if ((this.currentStep + 1) % this.workday === 0) {
      // Time to relax
      this.activity = States.HOME
      thinkTime = thinkTime * 5
    } else if (diceRoll < 0.10) {
      // 10% chance of getting coffee
      this.location = Office.coffeeShop(this.site)
      this.activity = States.COFFEE
    } else if (diceRoll < 0.15) {
      // 5% chance of lunchtime
      this.location = Office.cafeteria(this.site)
      this.activity = States.LUNCH
    } else if (diceRoll < 0.40) {
      // 25% chance of going to a meeting
      this.location = Office.randomOffice(this.site)
      this.activity = States.MEETING
    } else {
      // If all else fails, we will work ;)
      this.location = Office.randomOffice(this.site)
      this.activity = States.WORKING
    }
    if (this.location !== undefined && !await actor.call(this, actor.proxy('Office', this.location), 'isEmpty')) {
      // If the office we are going to next is non-empty, we will spend more time there.
      thinkTime = thinkTime * 2
    }
    this.currentStep = this.currentStep + 1

    // TODO: atomic checkpoint & doNext
    await this.checkpointState()
    await actor.tell(this, 'reportDecision', thinkTime, priorActivity)
  }

  // Update derived simulation state by informing other actors of our next action.
  // Schedule a reminder for the next move step.
  // Invoked as continuation from determineNextStep
  async reportDecision (thinkTime, priorActivity) {
    if (verbose) console.log(`${this.site}: ${this.name} will be doing ${this.activity} at ${this.location || 'off-site'} for ${thinkTime}ms`)
    if (this.location !== undefined) {
      await actor.call(this, actor.proxy('Office', this.location), 'enter', this.name)
    }
    if (this.activity !== priorActivity) {
      await actor.tell(actor.proxy('Site', this.site), 'workerUpdate', this.name, this.activity, Date.now())
    }
    const when = new Date(Date.now() + thinkTime)
    await actor.reminders.schedule(this, 'move', { id: 'step', targetTime: when }, when.getTime())
    if (debug) console.log(`${this.site}: Researcher ${this.name} completed move ${this.currentStep - 1}`)
  }
}

/*
 * Express / KAR boilerplate.  No application logic below here.
 */

const app = express()

app.post('/shutdown', async (_reg, res) => {
  console.log('Shutting down service')
  res.sendStatus(200)
  await sys.shutdown()
  server.close(() => process.exit())
})

app.use(sys.actorRuntime({ Company, Site, Office, Researcher }))

const server = sys.h2c(app).listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')
