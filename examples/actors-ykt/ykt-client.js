const { actor, shutdown, broadcast } = require('kar')

const sleep = (milliseconds) => {
  return new Promise(resolve => setTimeout(resolve, milliseconds))
}

const truthy = s => s && s.toLowerCase() !== 'false' && s !== '0'

async function testTermination (failure) {
  if (failure) {
    console.log('FAILED; setting non-zero exit code')
    process.exitCode = 1
  } else {
    console.log('SUCCESS')
    process.exitCode = 0
  }

  if (!truthy(process.env.KUBERNETES_MODE)) {
    console.log('Requesting server shutdown')
    await broadcast('shutdown')
  }

  console.log('Terminating sidecar')
  await shutdown()
}

async function main () {
  let failure = false

  const company = 'IBM'
  const researchDivision = {
    Yorktown: { workers: 20, thinkms: 2000, steps: 20 },
    Cambridge: { workers: 10, thinkms: 1000, steps: 40 },
    Almaden: { workers: 15, thinkms: 4000, steps: 10 }
  }

  console.log(`Staring simulation: ${JSON.stringify(researchDivision)}`)

  for (const site in researchDivision) {
    await actor.call('Site', site, 'resetDelayStats')
    await actor.call('Site', site, 'siteReport')
    await actor.call('Company', company, 'hire', Object.assign({ site }, researchDivision[site]))
  }

  while (true) {
    await sleep(5000)
    const totalWorking = await actor.call('Company', 'IBM', 'count')
    console.log(`Num working is ${totalWorking}`)
    // const sr = await actor.call('Site', 'Yorktown', 'siteReport')
    // console.log(sr)
    if (totalWorking === 0) {
      for (const site in researchDivision) {
        console.log(`Valiadating ${site}`)
        const delays = await actor.call('Site', site, 'delayReport')
        console.log(delays)
        const count = delays.delayHistogram.reduce((x, y) => x + y, 0)
        const expectedSteps = researchDivision[site].workers * researchDivision[site].steps
        if (count !== expectedSteps) {
          console.log(`FAILURE: Expected ${expectedSteps} moves but got ${count}`)
          failure = true
        }
      }
      break
    }
  }

  testTermination(failure)
}

main()
