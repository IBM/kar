const { actor } = require('kar')

const sleep = (milliseconds) => {
  return new Promise(resolve => setTimeout(resolve, milliseconds))
}

async function main () {
  const now = Date.now()
  const deadline = new Date(now + 10 * 1000)
  const period = '10s'
  const id = 'myTicker'

  await actor.scheduleReminder('A', '22', 'foo', { id, deadline })
  await sleep(20000)
}

main()
