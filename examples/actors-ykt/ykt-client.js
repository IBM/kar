const { actor } = require('kar')

const sleep = (milliseconds) => {
  return new Promise(resolve => setTimeout(resolve, milliseconds))
}

async function main () {
  await actor.call('Site', 'ykt', 'siteReport')
  await actor.call('Site', 'ykt', 'workDay', { workers: 1 })
  while (true) {
    await sleep(5000)
    await actor.call('Site', 'ykt', 'siteReport')
  }
}

main()
