const express = require('express')

const { logger, jsonParser, errorHandler, shutdown, actor, actors, actorRuntime, h2c } = require('kar')

const truthy = s => s && s.toLowerCase() !== 'false' && s !== '0'
const verbose = truthy(process.env.VERBOSE)
const debug = truthy(process.env.DEBUG)

function randI (max) { return Math.floor(Math.random() * Math.floor(max)) }

// The states a Researcher can be in
const States = {
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
      sites: this.sites
    }
    await this.sys.setMultiple(state)
  }

  async activate () {
    const state = await this.sys.getAll()
    this.nextSerialNumber = state.nextSerialNumber || 0
    this.sites = state.sites || []
  }

  async deactivate () {
    await this.checkpoint()
  }

  async hire ({ site = 'Yorktown', workers = 3, steps = 20, thinkms = 10000 } = {}) {
    if (!this.sites.includes(site)) {
      this.sites.push(site)
      actor.scheduleReminder('Site', site, 'siteReport', { id: 'aisle14', deadline: new Date(Date.now() + 1000), period: '5s' })
    }
    console.log(`${workers} hired to perform ${steps} tasks a day at ${site}`)
    const sn = this.nextSerialNumber
    this.nextSerialNumber = sn + workers
    await this.checkpoint()
    await this.sys.set('nextSerialNumber', this.nextSerialNumber)
    for (var i = 0; i < workers; i++) {
      const name = sn + i
      await actor.tell('Researcher', name, 'startDay', { site, steps, thinkms })
    }
  }

  async count () {
    let count = 0
    for (const site of this.sites) {
      count += await actors.Site[site].count()
    }
    return count
  }
}

// Sites update rapidly changing aggregate statistics by processing
// the workerUpdate and endWorkDay messages from individual workers
// Rather than checkpointing this data on every change, we will instead
// detect when a Site is being restored from a potentially outdated checkpoint
// and recompute the summary.
// TODO: This recovery is not yet implemented.
// Most likely scheme is that Company reliably
// tracks the names of all Researchers assigned to the site, so that a
// recovery can be implemented by getting the state of each of them to
// rebuild the workers object.
class Site {
  get count () { return Object.keys(this.workers).length }

  async checkpoint () {
    const state = {
      delayStats: this.delayStats,
      workers: this.workers
    }
    await this.sys.setMultiple(state)
  }

  async activate () {
    const state = await this.sys.getAll()
    this.delayStats = state.delayStats || []
    this.workers = state.workers || {}
    if (debug) console.log(`activated Site ${this.sys.id} with occupants ${state.workers}`)
  }

  async deactivate () {
    await this.checkpoint()
    if (debug) console.log(`deactivated Site ${this.sys.id}`)
  }

  async endWorkDay ({ who, delays = [] } = {}) {
    const ds = this.delayStats
    delays.forEach(function (missedMS, _) {
      const missedBucket = Math.floor(missedMS / delaysBucketMS)
      ds[missedBucket] = (ds[missedBucket] || 0) + 1
    })

    delete this.workers[who]
    if (debug) console.log(`${who} left Site ${this.sys.id} count is now ${this.count}`)
  }

  async workerUpdate ({ who, activity, office }) {
    this.workers[who] = activity
  }

  async siteReport () {
    const siteEmployees = this.count
    const time = new Date().toString()
    const status = { site: this.sys.id, siteEmployees, time }
    if (siteEmployees > 0) {
      for (const s in States) {
        status[States[s]] = 0
      }
      for (const worker in this.workers) {
        status[this.workers[worker]] += 1
      }
      if (siteEmployees > 0) {
        console.log(status)
      }
    }
    // TODO: publish siteReport to KAR pub-sub
    // await publish('siteReport', status)
    return status
  }

  delayReport () {
    for (const i in this.delayStats) {
      if (verbose) console.log(`${this.sys.id}: <${(parseInt(i) + 1) * delaysBucketMS}ms\t${this.delayStats[i]}`)
    }
    return { site: this.sys.id, bucketSizeInMS: delaysBucketMS, delayHistogram: this.delayStats }
  }

  resetDelayStats () {
    this.delayStats = []
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

  async checkpointState () {
    const state = {
      site: this.site,
      activity: this.activity,
      location: this.location,
      steps: this.steps,
      thinkms: this.thinkms,
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

  async updateActivity (newActivity) {
    if (this.activity !== newActivity) {
      await actor.tell('Site', this.site, 'workerUpdate', { who: this.name, activity: newActivity, office: this.location })
      this.activity = newActivity
    }
  }

  async startDay ({ site, steps, thinkms }) {
    this.site = site
    this.steps = steps
    this.thinkms = thinkms
    this.activity = States.COMMUTING
    await this.checkpointState()
    if (verbose) console.log(`${this.site}: ${this.name} is commuting to work`)

    await actor.tell('Site', this.site, 'workerUpdate', { who: this.name, activity: States.COMMUTING })

    const commuteTime = 1 + randI(5 * this.thinkms)
    const deadline = new Date(Date.now() + commuteTime)
    await this.sys.scheduleReminder('move', { id: 'move', deadline, data: deadline.getTime() })
  }

  async move (targetTime) {
    this.recordJitter(Math.abs(Date.now() - targetTime))
    if (debug) console.log(`${this.site}: Researcher ${this.name} entered move`)

    if (this.activity !== States.COMMUTING) {
      const oldLocation = this.location
      await actors.Office[oldLocation].leave(this.name)
    }

    this.steps = this.steps - 1
    if (this.steps <= 0) {
      await this.updateActivity(States.DONE)
      await actors.Site[this.site].endWorkDay({ who: this.name, delays: this.delays })
      await this.sys.delete('state') // FIXME:  Don't delete state when we enable multi-day simulations
      if (verbose) console.log(`${this.site}: Quitting time for ${this.name}`)
    } else {
      const nextMove = Math.random()
      let nextLocation
      let thinkTime = 1 + randI(this.thinkms)
      if (nextMove < 0.10) {
        // 10% chance of getting coffee
        nextLocation = Office.coffeeShop(this.site)
        if (verbose) console.log(`${this.site}: Coffee time for ${this.name}`)
        await this.updateActivity(States.COFFEE)
      } else if (nextMove < 0.15) {
        // 5% chance of lunchtime
        nextLocation = Office.cafeteria(this.site)
        thinkTime = thinkTime * 2
        if (verbose) console.log(`${this.site}: Lunch time for ${this.name}`)
        await this.updateActivity(States.LUNCH)
      } else {
        nextLocation = Office.randomOffice(this.site)
        if (!await actors.Office[nextLocation].isEmpty()) {
          thinkTime = thinkTime * 3
          if (verbose) console.log(`${this.site}: Researcher ${this.name} is heading to a meeting at ${nextLocation}`)
          await this.updateActivity(States.MEETING)
        } else {
          await this.updateActivity(States.WORKING)
        }
      }

      this.location = nextLocation
      const deadline = new Date(Date.now() + thinkTime)
      await this.sys.scheduleReminder('move', { id: 'move', deadline, data: deadline.getTime() })
      await this.checkpointState()

      await actors.Office[this.location].enter(this.name)
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
