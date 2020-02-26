const { async, sync } = require('./kar')

async function main () {
  console.log(await sync('myService', 'incr', 42))
}

main()
