const express = require('express')
const { actor, sys, publish } = require('kar')

// CloudEvents SDK for defining a structured HTTP request receiver.
const cloudevents = require('cloudevents-sdk/v1')

const app = express()

class SiteReportManager {
  get name () {
    return this.kar.id
  }

  async activate () {
    console.log('Activate Stock Manager')
    // TODO: Remove this, for dev only.
    actor.state.removeAll(this)
    const state = await actor.state.getAll(this)
    this.counter = state.counter || 0
    this.sites = state.sites || {}
  }

  async deactivate () {
    console.log('Deactivate Stock Manager')
    const state = {
      counter: this.counter,
      sites: this.sites
    }
    console.log(state)
    await actor.state.setMultiple(this, state)
  }

  async manageReport (reportEvent) {
    console.log(`New report event ${this.counter}`)
    var report = reportEvent.data

    // Add report to Company records.
    var sites = this.sites

    var changesPresent = false
    if (`${report.site}` in sites) {
      changesPresent = (report.siteEmployees !== sites[`${report.site}`])
    } else if (report.siteEmployees > 0) {
      changesPresent = true
    }

    // Record the new number of employees.
    sites[`${report.site}`] = report.siteEmployees
    console.log(sites)

    // For fire safety, we keep track of the people in the building. A quick
    // report is generated and printed to a Slack Channel. We only print a
    // reoport when there is a change in the number of employees of any site.
    // When all employees have departed we print a custom message.
    if (changesPresent) {
      var slackReportEvent = cloudevents.event()
        .type('employee.count')
        .source('ykt.aggregator')

      // Compose message.
      var slackMessage = 'Employee count: '
      var onSiteEmployees = false
      for (var key in sites) {
        if (sites[key] > 0) {
          slackMessage += ` ${key}: ${sites[key]} `
          onSiteEmployees = true
        }
      }

      // If there are no employees left anywhere, print special message.
      if (!onSiteEmployees) {
        slackMessage = 'End of work day. No on-site employees.'
      }

      // Add message to Cloud Event data field.
      slackReportEvent.data(slackMessage)

      // Publish event.
      publish('outputReport', slackReportEvent)
    }

    // Increment the counter and return.
    this.counter += 1
    return this.counter
  }
}

// Subscribe the `manageReport` method of the SiteReportManager Actor to respond to events
// emitted on the 'siteReport' topic.
actor.subscribe(actor.proxy('SiteReportManager', 'Reports'), 'siteReport', 'manageReport')

// Enable actor.
app.use(sys.actorRuntime({ SiteReportManager }))

// Boilerplate code for terminating the service.
app.post('/shutdown', async (_reg, res) => {
  console.log('Shutting down service')
  res.sendStatus(200)
  await sys.shutdown()
  server.close(() => process.exit())
})

// Enable kar error handling.
app.use(sys.errorHandler)

const server = app.listen(process.env.KAR_APP_PORT, '127.0.0.1')
