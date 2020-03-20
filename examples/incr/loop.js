const { broadcast, shutdown, call } = require('kar')

async function main () {
  let x = 0
  var failure = false
  console.log('Initiating 500 sequential increments')
  for (let i = 0; i < 500; i++) {
    x = await call('myService', 'incrQuiet', x)
    if (i % 100 == 0) { console.log(`incr(${i} = ${x})`) }
    if (x !== i + 1) {
      console.log(`Failed! incr(${i}) returned ${x}`)
      failure = true
    }
  }
  console.log('Sequential increments completed')

  console.log('Initiating 250 potentially concurrent increments')
  const incs = Array.from(new Array(250), (_, i) => i + 1000).map(function (elem, _) {
    return call('myService', 'incrQuiet', elem)
      .then(function (v) {
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

  if (failure) {
    console.log('FAILED; setting non-zero exit code')
    process.exitCode = 1
  } else {
    console.log('SUCCESS')
    process.exitCode = 0
  }

  console.log('Requesting server shutdown')
  await broadcast('shutdown')

  console.log('Terminating sidecar')
  await shutdown()
}

main()
