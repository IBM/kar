const { actor } = require('kar')

const sleep = (milliseconds) => {
  return new Promise(resolve => setTimeout(resolve, milliseconds))
}

async function main () {
  const now = Date.now()
  const deadline = new Date(now + 3 * 1000)
  const period = '5s'

  await actor.scheduleReminder('A', '22', 'foo', { id: 'ticker', deadline, period })
  await actor.scheduleReminder('A', '22', 'foo', { id: 'once', deadline })
  await sleep(20000)
}

main()
