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

  const params = { workers: 10, thinkms: 2000, steps: 20 }
  await actor.call('Site', 'ykt', 'resetDelayStats')
  await actor.call('Site', 'ykt', 'siteReport')
  console.log(`Staring YKT simulation: ${JSON.stringify(params)}`)
  await actor.call('Site', 'ykt', 'workDay', params)
  while (true) {
    await sleep(5000)
    const report = await actor.call('Site', 'ykt', 'siteReport')
    console.log(`Num working is ${report.totalWorking}`)
    if (report.totalWorking === 0) {
      const delays = await actor.call('Site', 'ykt', 'delayReport')
      console.log(delays)
      const count = delays.delayHistogram.reduce((x, y) => x + y, 0)
      if (count !== params.workers * params.steps) {
        console.log(`FAILURE: Expected ${params.workers * params.steps} moves but got ${count}`)
        failure = true
      }
      break
    }
  }

  testTermination(failure)
}

main()
