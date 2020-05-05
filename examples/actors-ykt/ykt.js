const express = require('express')

const { logger, jsonParser, errorHandler, shutdown, actor, actorRuntime, h2c } = require('kar')

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
  }

  async deactivate () {
    await this.checkpoint()
  }

  get count () { return Object.keys(this.bluepages).length }

  activeEmployees ({ site }) {
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
      await actor.call(this, actor.proxy('Site', site), 'joinCompany', { company: this.name })
      await actor.reminders.schedule(actor.proxy('Site', site), 'siteReport', { id: 'aisle14', deadline: new Date(Date.now() + 1000), period: '5s' })
      await actor.state.set(this, 'sites', this.sites)
    }

    console.log(`${workers} hired to perform ${steps} tasks/day for ${days} days at ${site}`)

    const sn = this.nextSerialNumber
    this.nextSerialNumber = sn + workers
    await actor.state.set(this, 'nextSerialNumber', this.nextSerialNumber)

    for (var i = 0; i < workers; i++) {
      const name = sn + i
      this.bluepages[name] = site
      await actor.tell(actor.proxy('Site', site), 'newHire', { who: name, days, steps, thinkms })
    }
    await actor.state.set(this, 'bluepages', this.bluepages)
  }

  async retire ({ who }) {
    delete this.bluepages[who]
    await actor.state.set(this, 'bluepages', this.bluepages)
    await actor.state.removeAll(actor.proxy('Researcher', who))
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
      const employees = await actor.call(actor.proxy('Company', this.company), 'activeEmployees', { site: this.name })
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

  async joinCompany ({ company }) {
    this.company = company
    await actor.state.set(this, 'company', company)
  }

  async newHire ({ who, days, steps, thinkms }) {
    this.workers[who] = States.ONBOARDING
    await actor.state.set(this, 'workers', this.workers)
    await actor.tell(actor.proxy('Researcher', who), 'newHire', { site: this.name, days, steps, thinkms })
  }

  async retire ({ who, delays = [] }) {
    if (this.workers[who]) {
      const ds = this.reminderDelays
      delays.forEach(function (missedMS, _) {
        const missedBucket = Math.floor(missedMS / bucketSizeInMS)
        ds[missedBucket] = (ds[missedBucket] || 0) + 1
      })
      delete this.workers[who]
      await actor.state.setMultiple(this, { workers: this.workers, reminderDelays: this.reminderDelays })
    }

    await actor.call(this, actor.proxy('Company', this.company), 'retire', { who })
    if (verbose) console.log(`Researcher ${who} has retired. Site employee count is now ${this.count}`)
  }

  async workerUpdate ({ who, activity, office, timestamp }) {
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
    // TODO: publish siteReport event to company channel via KAR pub-sub
    // await publish('siteReport', status)
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

class Researcher {
  get name () { return this.kar.id }

  async newHire ({ site, days, steps, thinkms }) {
    const initialState = { site, career: days * steps, workday: steps, thinkms, activity: States.ONBOARDING, delays: [] }
    Object.assign(this, initialState)
    await actor.state.setMultiple(this, initialState)

    const deadline = new Date(Date.now() + thinkms)
    await actor.reminders.schedule(this, 'move', { id: 'step', deadline, data: deadline.getTime() })
  }

  // Checkpoint only saves the transitory state of the Researcher
  // All initialize-only fields are persisted in newHire.
  async checkpointState () {
    if (this.retired) return
    const state = {
      activity: this.activity,
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

    if (debug) console.log(`${this.site}: Researcher ${this.name} entered move with delay ${observedDelay}`)

    // Clear derived simulation state (idempotent)
    if (this.location !== undefined) {
      const oldLocation = this.location
      await actor.call(this, actor.proxy('Office', oldLocation), 'leave', this.name)
    }

    // Is it time to retire?
    if (this.delays.length === this.career) {
      this.retired = true
      await actor.call(this, actor.proxy('Site', this.site), 'retire', { who: this.name, delays: this.delays })
      return
    }

    // Still an active Researcher. What to do next?
    const stepNumber = this.delays.length
    const diceRoll = Math.random()
    let nextLocation
    let nextActivity
    let thinkTime = 1 + randI(this.thinkms)
    if (stepNumber % this.workday === 0) {
      // Morning rush hour.
      nextActivity = States.COMMUTING
      thinkTime = thinkTime * 3
    } else if ((stepNumber + 2) % this.workday === 0) {
      // Evening rush hour.
      nextActivity = States.COMMUTING
      thinkTime = thinkTime * 2
    } else if ((stepNumber + 1) % this.workday === 0) {
      // Time to relax
      nextActivity = States.HOME
      thinkTime = thinkTime * 5
    } else if (diceRoll < 0.10) {
      // 10% chance of getting coffee
      nextLocation = Office.coffeeShop(this.site)
      nextActivity = States.COFFEE
    } else if (diceRoll < 0.15) {
      // 5% chance of lunchtime
      nextLocation = Office.cafeteria(this.site)
      nextActivity = States.LUNCH
    } else if (diceRoll < 0.40) {
      // 25% chance of going to a meeting
      nextLocation = Office.randomOffice(this.site)
      nextActivity = States.MEETING
    } else {
      // If all else fails, we will work ;)
      nextLocation = Office.randomOffice(this.site)
      nextActivity = States.WORKING
    }
    if (nextLocation !== undefined && !await actor.call(this, actor.proxy('Office', nextLocation), 'isEmpty')) {
      // If our nextLocation is non-empty, we will spend more time there.
      thinkTime = thinkTime * 2
    }

    // Report our intent; this tell is non-definitive if there is a failure before we commit.
    if (verbose) console.log(`${this.site}: ${this.name} will be doing ${nextActivity} at ${nextLocation || 'off-site'}`)
    if (this.activity !== nextActivity) {
      await actor.tell(actor.proxy('Site', this.site), 'workerUpdate',
        { who: this.name, activity: nextActivity, office: nextLocation, timestamp: Date.now() })
    }

    // Commit the decision by checkpointing our state.
    this.activity = nextActivity
    this.delays.push(observedDelay)
    this.location = nextLocation
    await this.checkpointState()

    // Update derived simulation state. Must happen after checkpoint to ensure
    // that even if we suffer a failure we will always call leave on the office before we retire.
    if (this.location !== undefined) {
      await actor.call(this, actor.proxy('Office', this.location), 'enter', this.name)
    }

    // Schedule next step
    const deadline = new Date(Date.now() + thinkTime)
    await actor.reminders.schedule(this, 'move', { id: 'step', deadline, data: deadline.getTime() })

    if (debug) console.log(`${this.site}: Researcher ${this.name} exited move`)
  }
}

/*
 * Express / KAR boilerplate.  No application logic below here.
 */

const app = express()

app.use(logger, jsonParser) // enable kar logging and parsing

app.post('/shutdown', async (_reg, res) => {
  console.log('Shutting down service')
  res.sendStatus(200)
  await shutdown()
  server.close(() => process.exit())
})

app.use(actorRuntime({ Company, Site, Office, Researcher }))
app.use(errorHandler)
const server = h2c(app).listen(process.env.KAR_APP_PORT, '127.0.0.1')
