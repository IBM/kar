const { actor } = require('kar')

const sleep = (milliseconds) => {
  return new Promise(resolve => setTimeout(resolve, milliseconds))
}

async function main () {
  await actor.call('Site', 'ykt', 'siteReport')
  await actor.call('Site', 'ykt', 'workDay', { workers: 10 })
  while (true) {
    await sleep(10000)
    const report = await actor.call('Site', 'ykt', 'siteReport')
    if (report.totalWorking === 0) {
      const delays = await actor.call('Site', 'ykt', 'delayReport')
      console.log(delays)
      break
    }
  }
}

main()
