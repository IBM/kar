const { actor, sys } = require('kar')

const sleep = (milliseconds) => {
  return new Promise(resolve => setTimeout(resolve, milliseconds))
}

async function testTermination (failure) {
  if (failure) {
    console.log('FAILED; setting non-zero exit code')
    process.exitCode = 1
  } else {
    console.log('SUCCESS')
    process.exitCode = 0
  }

  console.log('Terminating sidecar')
  await sys.shutdown()
}

async function main () {
  let failure = false
  let countdown = 60
  const cafe = actor.proxy('Cafe', 'Cafe de Flore')

  console.log('Serving a meal:')
  const table = await actor.call(cafe, 'seatTable', 20, 5)

  let occupancy = 1
  while (occupancy > 0 & !failure) {
    occupancy = await actor.call(cafe, 'occupancy', table)
    console.log(`Table occupancy is ${occupancy}`)
    await sleep(2000)
    countdown = countdown - 1
    if (countdown < 0) {
      console.log('TOO SLOW: Countdown reached 0!')
      failure = true
    }
  }

  testTermination(failure)
}

main()
