const { actor } = require('kar')

async function main () {
  console.log(await actor.call('myService', 'myInstance', 'incr', 42))
}

main()
