const { actor } = require('kar')

const sleep = (milliseconds) => {
  return new Promise(resolve => setTimeout(resolve, milliseconds))
}

async function main () {
  const now = Date.now()
  const targetTime = new Date(now + 3 * 1000)
  const period = '5s'
  const a22 = actor.proxy('Foo', '22')
  const a23 = actor.proxy('Foo', '23')
  const a2112 = actor.proxy('Foo', '2112')
  await actor.reminders.schedule(a22, 'echo', { id: 'ticker', targetTime, period }, 'hello', 'my friend', 'my foe')
  await actor.reminders.schedule(a23, 'echo', { id: 'ticker', targetTime, period })
  await actor.reminders.schedule(a2112, 'echo', { id: 'ticker', targetTime, period }, 'Syrinx')
  await actor.reminders.schedule(a22, 'echo', { id: 'once', targetTime }, 'carpe diem')
  console.log(await actor.reminders.get(a23))
  console.log(await actor.reminders.get(a22, 'noone'))
  console.log(await actor.reminders.get(a22, ''))
  console.log(await actor.reminders.get(a22, 'ticker'))
  await sleep(20000)
  await actor.reminders.cancel(a22, 'ticker')
  await actor.reminders.cancel(a2112)
  await sleep(20000)
}

main()
