const { actor } = require('kar')

async function main () {
  // actor state
  await actor.state.set('foo', 'myInstance', 'key1', 42)
  await actor.state.set('foo', 'myInstance', 'key2', 'abc123')
  await actor.state.set('foo', 'myInstance', 'key3', { field: 'value' })
  await actor.state.set('foo', 'myInstance', 'key4', null)
  console.log(await actor.state.get('foo', 'myInstance', 'key1'))
  console.log(await actor.state.get('foo', 'myInstance', 'key2'))
  console.log(await actor.state.getAll('foo', 'myInstance'))
  await actor.state.delete('foo', 'myInstance', 'key2')
  console.log(await actor.state.getAll('foo', 'myInstance'))
  await actor.state.deleteAll('foo', 'myInstance')
  console.log(await actor.state.getAll('foo', 'myInstance'))

  console.log(await actor.call('foo', 'myInstance', 'incr', 42))
}

main()
