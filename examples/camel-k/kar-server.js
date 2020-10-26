const express = require('express')
const { actor, sys, events } = require('kar')

// CloudEvents SDK for defining a structured HTTP request receiver.
const { CloudEvent } = require('cloudevents')

const app = express()

class StockManager {
  get name () {
    return this.kar.id
  }

  async activate () {
    console.log('Activate Stock Manager')
    // TODO: Remove this, for dev only.
    actor.state.removeAll(this)
    const state = await actor.state.getAll(this)
    this.counter = state.counter || 0
    this.package = state.package || {}
    this.maxPrice = state.maxPrice || -1
  }

  async deactivate () {
    console.log('Deactivate Stock Manager')
    const state = {
      counter: this.counter,
      package: this.package,
      maxPrice: this.maxPrice
    }
    console.log(state)
    await actor.state.setMultiple(this, state)
  }

  async manage (stockPriceEvent) {
    console.log('KAR stock price manager: Cloud Event Received')

    // In this particular case we know the encoding type of the data and we
    // can therefore decode it accordingly. In general, to allow the communication
    // via Cloud Events, the publisher and the subscriber must agree on the encoding
    // of the payload.

    // TODO: Add support for more data types.
    var data
    if (stockPriceEvent.data_base64 !== undefined) {
      data = Buffer.from(stockPriceEvent.data_base64, 'base64').toString()
    } else {
      console.error('Unexpected data type.')
    }

    // Log data.
    console.log(data)

    // In addition to the deserialization of the data there are cases in which the
    // payload can be parsed further. In this case we will parse the data as JSON.

    // In this case we need to make valid JSON out of the payload by adding quotes.
    var dataJson = JSON.parse(data.replace(/([a-zA-Z0-9-]+)=([a-zA-Z0-9-.]+)/g, '"$1":"$2"'))

    // Create an entry in the list of stock prices.
    this.package[`stock_${this.counter}`] = {}
    const stock = this.package[`stock_${this.counter}`]
    stock.name = dataJson[0].symbol
    var currentPrice = parseFloat(dataJson[0].price)
    stock.price = currentPrice

    // Compute the new maximum price.
    var newHigh = false
    if (this.maxPrice < currentPrice) {
      this.maxPrice = currentPrice
      newHigh = true
    }

    // Show state of stock prices.
    console.log(this.package)

    // Prepare to send information to Slack. All that is required is the publication
    // of the desried output string on the OutputStockEvent topic.
    // Send the event to Slack if the price has increased from the previous reading.
    if (this.counter > 1) {
      // Read previous value.
      var prevCounter = this.counter - 1
      var previousPrice = this.package[`stock_${prevCounter}`].price
      var increase = currentPrice - previousPrice
      if (increase >= 0) {
        var userInfo = `Price of ${stock.name} has increased by ${increase} to ${currentPrice}.`

        if (newHigh) {
          userInfo += ' Stock has reached a new high this session. Sell, sell, sell!'
        }

        // Create a CloudEvent to hold to result.
        // Note: data_base64 is not available in the cloud events javascript SDK as is available in the
        // Java SDK. We use the plain data field. The Java Cloud Event deserialization can handle
        // of this case.
        var slackStockEvent = new CloudEvent({
          type: 'stock.output',
          source: 'kar.processor',
          data: userInfo
        })

        // Publish event.
        events.publish('OutputStockEvent', slackStockEvent)
      }
    }

    // Increment the counter and return.
    this.counter += 1
    return this.counter
  }
}

// Subscribe the `manage` method of the StockManager Actor to respond to events emitted on
// the 'InputStockEvent' topic.
events.subscribe(actor.proxy('StockManager', 'ITStocks'), 'manage', 'InputStockEvent')

// Enable actor.
app.use(sys.actorRuntime({ StockManager }))

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
