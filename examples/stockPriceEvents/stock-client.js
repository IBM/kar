const { call } = require('kar')
const yargs = require('yargs')

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
  console.log(await call('price-sender', `stockprice/${stockName}`, ''))
}

main()
