const { actor } = require('kar')

const sleep = (milliseconds) => {
  return new Promise(resolve => setTimeout(resolve, milliseconds))
}

async function main () {
  const now = Date.now()
  const deadline = new Date(now + 3 * 1000)
  const period = '5s'

  await actor.scheduleReminder('Foo', '22', '/echo', { id: 'ticker', deadline, period, data: { msg: 'hello' } })
  await actor.scheduleReminder('Foo', '23', '/echo', { id: 'ticker', deadline, period })
  await actor.scheduleReminder('Foo', '2112', '/echo', { id: 'ticker', deadline, period, data: { msg: 'Syrinx' } })
  await actor.scheduleReminder('Foo', '22', '/echo', { id: 'once', deadline, data: { msg: 'carpe diem' } })
  console.log(await actor.getReminder('Foo', '23'))
  console.log(await actor.getReminder('Foo', '22', { id: 'noone' }))
  console.log(await actor.getReminder('Foo', '22', { id: '' }))
  console.log(await actor.getReminder('Foo', '22', { id: 'ticker' }))
  await sleep(20000)
  await actor.cancelReminder('Foo', '22', { id: 'ticker' })
  await actor.cancelReminder('Foo', '2112')
  await sleep(20000)
}

main()
