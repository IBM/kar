const { actor } = require('kar')

async function main () {
  console.log('parallel test starting')
  await actor.call(actor.proxy('Par', 1), 'test', process.argv[2] || 6)
  console.log('parallel test finished')
  process.exit(0)
}

main()
