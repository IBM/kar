const { async, shutdown, sync } = require('./kar')

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

  await async('myService', 'shutdown')

  await shutdown()

  if (failure) {
    console.log('Test failure; setting non-zero exit code')
    process.exitCode = 1
  } else {
    console.log('All tests succeeded')
  }
}

main()
