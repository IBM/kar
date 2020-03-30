const { actors } = require('kar')

async function main () {
  const a = actors.Foo[123]

  // actor state
  await a.sys.set('key1', 42)
  await a.sys.set('key2', 'abc123')
  await a.sys.set('key3', { field: 'value' })
  await a.sys.set('key4', null)
  console.log(await a.sys.get('key1'))
  console.log(await a.sys.get('key2'))
  console.log(await a.sys.getAll())
  await a.sys.delete('key2')
  console.log(await a.sys.getAll())
  await a.sys.deleteAll()
  console.log(await a.sys.getAll())

  // synchronous invocation
  console.log(await a.incr(42))

  // asynchronous invocation
  console.log(await a.sys.tell('incr', 42))

  // getter
  console.log(await a.field())

  // error in synchronous invocation
  try {
    console.log(await a.fail('error message 123'))
  } catch (err) {
    console.log('caught', err)
  }

  // undefined method
  try {
    console.log(await a.missing('error message 123'))
  } catch (err) {
    console.log('caught', err)
  }
}

main()
