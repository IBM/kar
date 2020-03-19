const { call } = require('kar')

async function main () {
  console.log(await call('myService', 'incr', 42))
}

main()
