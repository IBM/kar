const { broadcast, shutdown, sync } = require('./kar')

async function main () {
  let x = 0
  var failure = false
  console.log('Initiating 500 sequential increments')
  for (let i = 0; i < 500; i++) {
    x = await sync('myService', 'incr', x)
    if (x !== i + 1) {
      console.log(`Failed! incr(${i}) returned ${x}`)
      failure = true
    }
  }
  console.log('Sequential increments completed')

  console.log('Initiating 50 potentially concurrent increments')
  const incs = Array.from(new Array(50), (_, i) => i + 1000).map(function (elem, _) {
    return sync('myService', 'incr', elem)
      .then(function (v) {
        console.log(`${v}`)
        if (v !== elem + 1) {
          return Promise.reject(new Error(`Failed! incr(${elem}) returned ${v}`))
        } else {
          return Promise.resolve(`Success incr ${elem} returned ${v}`)
        }
      })
  })
  await Promise.all(incs)
    .then(function (_) {
      console.log('All concurrent increments completed successfully')
    })
    .catch(function (reason) {
      console.log(reason)
      failure = true
    })

  console.log('Requesting server shutdown')
  await broadcast('shutdown')

  console.log('Terminating sidecar')
  await shutdown()

  if (failure) {
    console.log('Test failure; setting non-zero exit code')
    process.exitCode = 1
  } else {
    console.log('All tests succeeded')
    process.exitCode = 0
  }
}

main()
