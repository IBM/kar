const express = require('express')
const https = require('https')
var path = require('path')
const { events, sys } = require('kar')

// CloudEvents SDK for constructing the event.
const v1 = require('cloudevents-sdk/v1')

const app = express()

app.use(express.json({ strict: false }))
app.use(express.json({ type: 'application/cloudevents+json' }))
app.use(express.urlencoded({ extended: true }))
app.use(express.static(path.join(__dirname, 'public')))

// Define the route this service provides
app.post('/stockprice/:stock_name', (req, res) => {
  let stocks = ''
  const stockName = req.params.stock_name

  const occurenceConfig = {
    hostname: 'financialmodelingprep.com',
    port: 443,
    path: '/api/v3/historical-price-full/' + stockName + '?apikey=demo',
    method: 'GET'
  }

  // Stock Price CloudEvent
  const stockEvent = v1.event()
    .type('stock.event')
    .source('financialmodelingprep.com')

  const stockPriceReq = https.request(occurenceConfig, (httpResponse) => {
    httpResponse.on('data', (chunk) => {
      const msg = `Getting stock price for ${stockName}.`
      console.log(msg)
      stocks += chunk
    })

    httpResponse.on('end', () => {
      var stockDataList = JSON.parse(stocks)
      var openPrices = stockDataList.historical.map(function (item) {
        return item.open
      })

      const msg = openPrices[openPrices.length - 1].toString()
      console.log(msg)

      // Set data of cloud event to stock data.
      stockEvent.data(openPrices.toString())

      // Send CloudEvent on the 'historical-prices' topic.
      events.publish('historical-prices', stockEvent)

      res.send(msg)
    })
  })

  stockPriceReq.on('error', (error) => {
    console.log(error)
  })

  stockPriceReq.end()
})

// Enable kar error handling.
app.use(sys.errorHandler)

app.listen(process.env.KAR_APP_PORT, '127.0.0.1')
