const { actor } = require('./kar')

async function main () {
  console.log(await actor.sync('myService', 'myInstance', 'incr', 42))
}

main()
