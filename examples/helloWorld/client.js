const { call } = require('kar')

async function main () {
  console.log(await call('greeter', 'hello', 'John Doe'))
}

main()
