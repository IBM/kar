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

const delaysBucketMS = 100

class Company {
  async checkpoint () {
    const state = {
      nextSerialNumber: this.nextSerialNumber,
      sites: this.sites,
      bluepages: this.bluepages
    }
    await this.sys.setMultiple(state)
  }

  async activate () {
    const state = await this.sys.getAll()
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
      await actor.call('Site', site, 'joinCompany', { company: this.sys.id })
      await actor.scheduleReminder('Site', site, 'siteReport', { id: 'aisle14', deadline: new Date(Date.now() + 1000), period: '5s' })
      await this.sys.set('sites', this.sites)
    }

    console.log(`${workers} hired to perform ${steps} tasks/day for ${days} days at ${site}`)

    const sn = this.nextSerialNumber
    this.nextSerialNumber = sn + workers
    await this.sys.set('nextSerialNumber', this.nextSerialNumber)

    for (var i = 0; i < workers; i++) {
      const name = sn + i
      this.bluepages[name] = site
      await actor.tell('Site', site, 'newHire', { who: name, days, steps, thinkms })
    }
    await this.sys.set('bluepages', this.bluepages)
  }

  async retire ({ who }) {
    delete this.bluepages[who]
    await this.sys.set('bluepages', this.bluepages)
    await actor.state.deleteAll('Researcher', who)
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
  get count () { return Object.keys(this.workers).length }

  async checkpoint () {
    const state = {
      reminderDelays: this.reminderDelays,
      workers: this.workers
    }
    await this.sys.setMultiple(state)
  }

  async activate () {
    const state = await this.sys.getAll()
    this.reminderDelays = state.reminderDelays || []
    this.workers = state.workers || {}
    this.company = state.company
    await this.sys.set('cleanShutdown', false)
    if (this.company !== undefined && state.cleanShutdown !== true) {
      const employees = await actor.call('Company', this.company, 'activeEmployees', { site: this.sys.id })
      for (const sn of employees) {
        this.workers[sn] = States.ONBOARDING // Not accurate, but will be corrected on next workerUpdate
      }
    }
    if (debug) console.log(`activated Site ${this.sys.id} with occupants ${state.workers}`)
  }

  async deactivate () {
    await this.checkpoint()
    await this.sys.set('cleanShutdown', true)
    if (debug) console.log(`deactivated Site ${this.sys.id}`)
  }

  async joinCompany ({ company }) {
    this.company = company
    await this.sys.set('company', company)
    if (debug) console.log(`Site ${this.sys.id} joined Company ${this.company}`)
  }

  async newHire ({ who, days, steps, thinkms }) {
    this.workers[who] = States.ONBOARDING
    await actor.tell('Researcher', who, 'newHire', { site: this.sys.id, days, steps, thinkms })
  }

  async retire ({ who, delays = [] }) {
    const ds = this.reminderDelays
    delays.forEach(function (missedMS, _) {
      const missedBucket = Math.floor(missedMS / delaysBucketMS)
      ds[missedBucket] = (ds[missedBucket] || 0) + 1
    })
    this.sys.set('reminderDelays', this.reminderDelays)

    await actor.call('Company', this.company, 'retire', { who })
    delete this.workers[who]
    if (verbose) console.log(`Researcher ${who} has retired. Site employee count is now ${this.count}`)
  }

  async workerUpdate ({ who, activity, office }) {
    if (this.workers[who] !== undefined) {
      this.workers[who] = activity
    }
  }

  async siteReport () {
    const siteEmployees = this.count
    const time = new Date().toString()
    const status = { site: this.sys.id, siteEmployees, time }
    status.delaysBucketMS = delaysBucketMS
    status.reminderDelays = this.reminderDelays
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

  delayReport () {
    for (const i in this.reminderDelays) {
      if (verbose) console.log(`${this.sys.id}: <${(parseInt(i) + 1) * delaysBucketMS}ms\t${this.reminderDelays[i]}`)
    }
    return { site: this.sys.id, bucketSizeInMS: delaysBucketMS, delayHistogram: this.reminderDelays }
  }

  resetDelayStats () {
    this.reminderDelays = []
  }
}

// An Office's Actor ID is of the form Site:Office
class Office {
  static coffeeShop (site) { return `${site}:OutTakes` }
  static cafeteria (site) { return `${site}:Cafeteria` }
  static randomOffice (site) {
    // TODO: Introduce site-specific office numbering patterns just for fun.
    const floor = randI(3)
    const aisle = 1 + randI(40)
    const office = 1 + randI(64)
    return `${site}:${aisle}-${floor}${office}`
  }

  isEmpty () { return this.occupants.size === 0 }
  get count () { return this.occupants.size }

  async checkpoint () {
    await this.sys.set('occupants', Array.from(this.occupants))
  }

  async activate () {
    const so = await this.sys.get('occupants') || []
    this.occupants = new Set(so)
    if (debug) console.log(`activated Office ${this.sys.id} with occupancy ${this.count}`)
  }

  async deactivate () {
    await this.checkpoint()
    if (debug) console.log(`deactivated Office ${this.sys.id}`)
  }

  async enter ({ who, checkpoint = true }) {
    this.occupants.add(who)
    if (checkpoint) {
      await this.checkpoint()
    }
    if (debug) console.log(`${who} entered Office ${this.sys.id} occupancy is now ${this.count}`)
  }

  async leave ({ who, checkpoint = true }) {
    this.occupants.delete(who)
    if (checkpoint) {
      await this.checkpoint()
    }
    if (debug) console.log(`${who} left Office ${this.sys.id} occupancy is now ${this.count}`)
  }
}

class Researcher {
  get name () { return this.sys.id }

  async newHire ({ site, days, steps, thinkms }) {
    const initialState = { site, days, workdaySteps: steps, thinkms, activity: States.ONBOARDING }
    Object.assign(this, initialState)
    await this.sys.setMultiple(initialState)
    await this.startWorkDay()
  }

  // Checkpoint only saves the transitory state of the Researcher
  // Initialize-only fields are persisted in newHire.
  async checkpointState () {
    const state = {
      days: this.days,
      steps: this.steps,
      activity: this.activity,
      location: this.location,
      delays: this.delays
    }
    await this.sys.setMultiple(state)
  }

  async activate () {
    Object.assign(this, await this.sys.getAll())
    if (debug) console.log(`activated Researcher ${this.name} with state `, this)
  }

  async deactivate () {
    await this.checkpointState()
    if (debug) console.log(`deactivated Researcher ${this.name}`)
  }

  recordJitter (ms) {
    if (this.delays === undefined) {
      this.delays = []
    }
    this.delays.push(ms)
  }

  async updateActivity (newActivity, newLocation) {
    if (verbose) console.log(`${this.site}: ${this.name} is now ${newActivity} at ${newLocation || 'off-site'}`)
    if (this.activity !== newActivity) {
      await actor.tell('Site', this.site, 'workerUpdate', { who: this.name, activity: newActivity, office: newLocation })
      this.activity = newActivity
    }
  }

  async startWorkDay () {
    await this.updateActivity(States.COMMUTING)
    this.steps = this.workdaySteps
    const commuteTime = 1 + randI(2 * this.thinkms)
    const deadline = new Date(Date.now() + commuteTime)
    await this.sys.scheduleReminder('move', { id: 'step', deadline, data: deadline.getTime() })
    await this.checkpointState()
  }

  async move (targetTime) {
    if (debug) console.log(`${this.site}: Researcher ${this.name} entered move`)
    this.recordJitter(Math.abs(Date.now() - targetTime))

    if (this.location !== undefined) {
      const oldLocation = this.location
      await actor.call('Office', oldLocation, 'leave', { who: this.name })
    }

    // Figure out what this Researcher will do next
    let nextCallback = 'move'
    let nextLocation
    let nextActivity
    let thinkTime = 1 + randI(this.thinkms)
    const diceRoll = Math.random()
    this.steps = this.steps - 1
    if (this.steps <= 0) {
      this.days = this.days - 1
      nextActivity = States.HOME
      nextCallback = 'startWorkDay'
      thinkTime = thinkTime + (5 * this.thinkms)
    } else if (this.steps === 1) {
      nextActivity = States.COMMUTING
    } else if (diceRoll < 0.10) {
      // 10% chance of getting coffee
      nextLocation = Office.coffeeShop(this.site)
      nextActivity = States.COFFEE
    } else if (diceRoll < 0.15) {
      // 5% chance of lunchtime
      nextLocation = Office.cafeteria(this.site, nextLocation)
      nextActivity = States.LUNCH
    } else if (diceRoll < 0.40) {
      // 25% chance of going to a meeting
      nextLocation = Office.randomOffice(this.site)
      nextActivity = States.MEETING
    } else {
      nextLocation = Office.randomOffice(this.site)
      nextActivity = States.WORKING
    }
    if (nextLocation !== undefined && !await actor.call('Office', nextLocation, 'isEmpty')) {
      // If our nextLocation is non-empty, we will spend more time there.
      thinkTime = thinkTime * 2
    }

    // Act on the decision. Trigger notifications, checkpoint state, schedule next step
    await this.updateActivity(nextActivity, nextLocation)
    this.location = nextLocation
    if (this.days > 0) {
      const deadline = new Date(Date.now() + thinkTime)
      await this.sys.scheduleReminder(nextCallback, { id: 'step', deadline, data: deadline.getTime() })
      await this.checkpointState()
    } else {
      await actor.call('Site', this.site, 'retire', { who: this.name, delays: this.delays })
    }

    if (this.location !== undefined) {
      await actor.call('Office', this.location, 'enter', { who: this.name })
    }

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
