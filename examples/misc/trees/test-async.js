const { actor } = require('kar')

async function main () {
  console.log('async test starting')
  await actor.call(actor.proxy('Async', 1), 'test', process.argv[2] || 6)
  console.log('async test finished')
  process.exit(0)
}

main()
