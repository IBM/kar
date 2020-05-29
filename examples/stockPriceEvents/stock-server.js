const express = require('express')
const { actor, sys, subscribe } = require('kar')

// CloudEvents SDK for defining a structured HTTP request receiver.
const cloudevents = require('cloudevents-sdk/v1')

const app = express()

// Subscribe to topic.
const topic = 'historical-prices'
// Subscribe service to topic.
// An option was added to specify the expected content type.
subscribe(topic, 'print-historical-prices', { contentType: 'application/cloudevents+json' })

app.use(express.json({ strict: false }))
app.use(express.json({ type: 'application/cloudevents+json' }))

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

class Portfolio {
  get name () {
    return this.kar.id
  }

  async activate () {
    console.log('Activate Portofolio')
    const state = await actor.state.getAll(this)
    this.counter = state.counter || 0
    this.package = state.package || {}
  }

  async deactivate () {
    console.log('Deactivate Portofolio')
    const state = {
      counter: this.counter,
      package: this.package
    }
    console.log(state)
    await actor.state.setMultiple(this, state)
  }

  async buy (buyStockEvent) {
    console.log('Cloud Event Received')
    // Only interested in the data field at this point since all the other
    // CloudEvent-specific fields have been checked. The other fields can still
    // be accessed here. For example, the type and source of the even may be
    // relevant to application logic.
    const purchaseData = buyStockEvent.data
    console.log(`Buy a batch of ${purchaseData.quantity} ${purchaseData.stock} stock.`)
    this.package[`stock_${this.counter}`] = {}
    const stock = this.package[`stock_${this.counter}`]
    stock.name = purchaseData.stock
    stock.quantity = purchaseData.quantity
    stock.price = purchaseData.price
    this.counter += 1

    // Show state of the Portfolio
    console.log(this.package)
    return this.counter
  }
}

// Subscribe the `buy` method of the Portfolio Actor to respong to events emitted on
// the 'buy-stock' topic.
actor.subscribe(actor.proxy('Portfolio', 'ITStocks'), 'buy-stock', 'buy')

// Enable actor.
app.use(sys.actorRuntime({ Portfolio }))

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
