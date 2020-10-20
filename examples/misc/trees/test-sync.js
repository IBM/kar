const { actor } = require('kar')

async function main () {
  console.log('sync test starting')
  await actor.call(actor.proxy('Sync', 1), 'test', process.argv[2] || 6)
  console.log('sync test finished')
  process.exit(0)
}

main()
