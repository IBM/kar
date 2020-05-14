const yargs = require('yargs')

// Retry http requests up to 10 times over 10s.
const fetch = require('fetch-retry')(require('node-fetch'), { retries: 10 })

// Request url for a KAR call service and route on that service.
function url (service, route) {
  return `http://127.0.0.1:${process.env.KAR_RUNTIME_PORT}/kar/v1/service/${service}/call/${route}`
}

async function main () {
  const argv = yargs
    .default({ stock: 'AAPL' })
    .usage('Usage: $0 --stock=[string]')
    .alias('stock', 's')
    .help()
    .alias('help', 'h')
    .argv

  var stockName = argv.stock

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
}

main()
