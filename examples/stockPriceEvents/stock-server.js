const express = require('express')
const { sys, subscribe } = require('kar')
// const { errorHandler } = require('kar')
// CloudEvents SDK for defining a structured HTTP request receiver.
const cloudevents = require('cloudevents-sdk/v1')

const app = express()

// Subscribe to topic.
const topic = 'historical-prices'
// Subscribe service to topic.
// An option was added to specify the expected content type.
subscribe(topic, 'print-historical-prices', { contentType: 'application/cloudevents+json' })

app.use(sys.logger, sys.jsonParser)

const HTTPCloudEventReceiver = new cloudevents.StructuredHTTPReceiver()

function cloudEventHandler (res, data, headers) {
  try {
    const myevent = HTTPCloudEventReceiver.parse(data, headers)

    console.log('Accepted event:')
    console.log(JSON.stringify(myevent.format(), null, 2))

    res.status(201).json(myevent.format())
  } catch (err) {
    console.error(err)
    res.status(415)
      .header('Content-Type', 'application/json')
      .send(JSON.stringify(err))
  }
}

// Process subscription.
app.post('/print-historical-prices', function (req, res) {
  cloudEventHandler(res, req.body, req.headers)
})

// Enable kar error handling.
app.use(sys.errorHandler)

app.listen(process.env.KAR_APP_PORT, '127.0.0.1')
