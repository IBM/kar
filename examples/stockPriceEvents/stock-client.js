const { actor, call, publish } = require('kar')
const yargs = require('yargs')
const cloudevents = require('cloudevents-sdk/v1')

// Retry http requests up to 10 times over 10s.
const fetch = require('fetch-retry')(require('node-fetch'), { retries: 10 })

// Request url for a KAR call service and route on that service.
function url (service, route) {
  return `http://127.0.0.1:${process.env.KAR_RUNTIME_PORT}/kar/v1/service/${service}/call/${route}`
}

// Stocks to consider and available cash for each one.
const stocks = ['AAPL', 'GOOG', 'IBM', 'NVDA', 'MSFT', 'FB']
const cash = 5000

async function main () {
  const argv = yargs
    .default({ stock: 'DEFAULT' })
    .usage('Usage: $0 --stock=[string]')
    .alias('stock', 's')
    .help()
    .alias('help', 'h')
    .argv

  var stockName = argv.stock

  // If user provides a stock name then only the latest price of that stock
  // will be printed. Otherwise we create a portofolio of stocks based on
  // a few selected technology stocks. The quantity purchased is equal to the
  // floor of ( 5000 / price_per_share ).
  if (stockName !== 'DEFAULT') {
    console.log(`Waiting for a price for ${stockName} using pub/sub: .... `)

    // Send http request with text/plain content type.
    // The message body is empty.
    const res = await fetch(url('price-sender', `stockprice/${stockName}`), {
      method: 'POST',
      body: '',
      headers: { 'Content-Type': 'text/plain' }
    })

    // Print respone.
    const response = await res.text()
    console.log(response)
  } else {
    // Create Portfolio actor.
    const portfolio = actor.proxy('Portfolio', 'ITStocks')

    // Clear current portofolio.
    await actor.state.removeAll(portfolio)

    // Construct new portfolio.
    var stock
    for (stock of stocks) {
      console.log(`Fetch price for ${stock}.`)

      // Using the SDK this explicit call could be rewritten as:
      const response = parseFloat(await call('price-sender', `stockprice/${stock}`, ''))
      const quantity = Math.floor(cash / response)
      // await actor.call(portfolio, 'buy', stock, quantity, response)
      var buyStockEvent = cloudevents.event()
        .type('stock.purchase.event')
        .source('javascript-client')

      // Payload contents.
      var purchaseData = {}
      purchaseData.stock = stock
      purchaseData.quantity = quantity
      purchaseData.price = response

      // Add purchase data to event payload.
      buyStockEvent.data(JSON.parse(JSON.stringify(purchaseData)))

      console.log(`Send purchase event for ${stock} (QTY = ${quantity}, PPS = ${response}).`)
      publish('buy-stock', buyStockEvent)
    }
  }
}

main()
