const { actor } = require('kar')

const sleep = (milliseconds) => {
  return new Promise(resolve => setTimeout(resolve, milliseconds))
}

async function main () {
  const now = Date.now()
  const deadline = new Date(now + 3 * 1000)
  const period = '5s'

  await actor.scheduleReminder('A', '22', 'foo/bar', { id: 'ticker', deadline, period })
  await actor.scheduleReminder('A', '23', 'foo/bar', { id: 'ticker', deadline, period })
  await actor.scheduleReminder('A', '2112', 'foo/bar', { id: 'ticker', deadline, period })
  await actor.scheduleReminder('A', '22', 'foo/baz', { id: 'once', deadline })
  console.log(await actor.getReminder('A', '23'))
  console.log(await actor.getReminder('A', '22', { id: 'noone' }))
  console.log(await actor.getReminder('A', '22', { id: '' }))
  console.log(await actor.getReminder('A', '22', { id: 'ticker' }))
  await sleep(20000)
  await actor.cancelReminder('A', '22', { id: 'ticker' })
  await actor.cancelReminder('A', '2112')
  await sleep(20000)
}

main()
