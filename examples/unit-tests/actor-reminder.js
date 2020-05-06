const { actor } = require('kar')

const sleep = (milliseconds) => {
  return new Promise(resolve => setTimeout(resolve, milliseconds))
}

async function main () {
  const now = Date.now()
  const deadline = new Date(now + 3 * 1000)
  const period = '5s'
  const a22 = actor.proxy('Foo', '22')
  const a23 = actor.proxy('Foo', '23')
  const a2112 = actor.proxy('Foo', '2112')
  await actor.reminders.schedule(a22, 'echo', 'ticker', deadline, period, 'hello', 'my friend', 'my foe')
  await actor.reminders.schedule(a23, 'echo', 'ticker', deadline, period)
  await actor.reminders.schedule(a2112, 'echo', 'ticker', deadline, period, 'Syrinx')
  await actor.reminders.schedule(a22, 'echo', 'once', deadline, undefined, 'carpe diem')
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
