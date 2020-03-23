const { actor } = require('kar')

async function main () {
  console.log(await actor.call('foo', 'myInstance', 'incr', 42))
}

main()
