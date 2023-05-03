/*
 * Copyright IBM Corporation 2020,2023
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
const { actor, sys, events } = require('kar-sdk')

// CloudEvents SDK for defining a structured HTTP request receiver.
const { CloudEvent } = require('cloudevents')

const app = express()

class SiteReportManager {
  get name () {
    return this.kar.id
  }

  async activate () {
    const state = await actor.state.getAll(this)
    this.counter = state.counter || 0
    this.sites = state.sites || {}
  }

  async deactivate () {
    const state = {
      counter: this.counter,
      sites: this.sites
    }
    await actor.state.setMultiple(this, state)
  }

  async manageReport (reportEvent) {
    var report = reportEvent.data
    console.log(reportEvent)
    console.log(`Event ${this.counter}: ${JSON.stringify(report)}`)
    // console.log(report)

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

      var slackReportEvent = new CloudEvent({
        type: 'employee.count',
        source: 'ykt.aggregator',
        data: slackMessage
      })

      // Publish event.
      events.publish('outputReport', slackReportEvent)
    }

    // Increment the counter and return.
    this.counter += 1
    return this.counter
  }
}

// Subscribe the `manageReport` method of the SiteReportManager Actor to respond to events
// emitted on the 'siteReport' topic.
events.subscribe(actor.proxy('SiteReportManager', 'Reports'), 'manageReport', 'siteReport')

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

const server = app.listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')
