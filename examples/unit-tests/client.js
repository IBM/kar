const { call, asyncCall, actor } = require('kar')

async function main () {
  // synchronous call
  console.log(await call('myService', 'incr', 42))

  // async call 1
  const f = await asyncCall('myService', 'incr', 22)

  // async call 2
  const f2 = await actor.asyncCall(actor.proxy('Foo', 123), 'incr', 42)

  // await callback 1
  console.log(await f())

  // await callback
  console.log(await f2())
}

main()
