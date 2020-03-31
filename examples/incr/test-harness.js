const { actor, actors, broadcast, shutdown, call } = require('kar')

const truthy = s => s && s.toLowerCase() !== 'false' && s !== '0'
const verbose = truthy(process.env.VERBOSE)

async function serviceTests () {
  let failure = false
  console.log('Initiating 500 sequential increments')
  for (let i = 0; i < 500; i++) {
    const x = await call('myService', 'incrQuiet', i)
    if (i % 100 === 0) { console.log(`incr(${i}) = ${x}`) }
    if (x !== i + 1) {
      console.log(`Failed! incr(${i}) returned ${x}`)
      failure = true
    }
  }
  console.log('Sequential increments completed')

  console.log('Initiating 250 potentially concurrent increments')
  const incs = Array.from(new Array(250), (_, i) => i + 1000).map(function (elem, _) {
    return call('myService', 'incrQuiet', elem)
      .then(function (v) {
        if (v !== elem + 1) {
          return Promise.reject(new Error(`Failed! incr(${elem}) returned ${v}`))
        } else {
          return Promise.resolve(`Success incr ${elem} returned ${v}`)
        }
      })
  })
  await Promise.all(incs)
    .then(function (_) {
      console.log('All concurrent increments completed successfully')
    })
    .catch(function (reason) {
      console.log(reason)
      failure = true
    })

  return failure
}

async function actorTests () {
  const a = actors.Foo[123]
  let failure = false

  console.log('Testing actor state operations')
  // actor state
  await a.sys.set('key1', 42)
  await a.sys.set('key2', 'abc123')
  await a.sys.set('key3', { field: 'value' })
  await a.sys.set('key4', null)

  const v1 = await a.sys.get('key1')
  if (v1 !== 42) {
    console.log(`Failed: get of key1 returned ${v1}`)
    failure = true
  }

  const v2 = await a.sys.get('key2')
  if (v2 !== 'abc123') {
    console.log(`Failed: get of key2 returned ${v2}`)
    failure = true
  }

  const v3 = await a.sys.getAll()
  try {
    if (v3.key1 !== 42 ||
    v3.key2 !== 'abc123' ||
    v3.key3.field !== 'value' ||
    v3.key4 != null) {
      console.log(`Failed: getAll ${v3}`)
      failure = true
    }
  } catch (err) {
    console.log(`Failed during validation of getAll: ${err}.`)
    console.log(`    value was ${v3}`)
    failure = true
  }

  await a.sys.delete('key2')
  const v4 = await a.sys.getAll()
  if (v4.key2) {
    console.log(`Failed to delete key2: ${v4}`)
    failure = true
  }

  await a.sys.deleteAll()
  const v5 = await a.sys.getAll()
  if (Object.keys(v5).length !== 0) {
    console.log(`Failed to delete all keys: ${v5}`)
    failure = true
  }

  console.log('Testing actor invocation')

  // external synchronous invocation of an actor method
  for (let i = 0; i < 25; i++) {
    console.log(`starting ${i}`)
    const x = await actor.call('Foo', 'anotherInstance', 'incr', i)
    console.log(`   got back ${x}`)
    if (x !== i + 1) {
      console.log(`Failed! incr(${i}) returned ${x}`)
      failure = true
    }
  }

  // synchronous invocation via the actor
  const v6 = await a.incr(42)
  if (v6 !== 43) {
    console.log(`Failed: unexpected result from incr ${v6}`)
    failure = true
  }

  // asynchronous invocation via the actor
  const v8 = await a.sys.tell('incr', 42)
  if (v8 !== 'OK') {
    console.log(`Failed: unexpected result from tell ${v8}`)
    failure = true
  }

  // getter
  const v7 = await a.field()
  if (v7 !== 42) {
    console.log(`Failed: getter of 'field' returned ${v7}`)
    failure = true
  }

  console.log('Testing actor invocation error handling')
  // error in synchronous invocation
  try {
    console.log(await a.fail('error message 123'))
    console.log('Failed to raise expected error')
    failure = true
  } catch (err) {
    if (verbose) console.log('Caught expected error: ', err.message)
  }

  // undefined method
  try {
    console.log(await a.missing('error message 123'))
    console.log('Failed. No error raised invoking missing method')
    failure = true
  } catch (err) {
    if (verbose) console.log('Caught expected error: ', err.message)
  }

  return failure
}

async function main () {
  var failure = false

  console.log('*** Service Tests ***')
  failure |= await serviceTests()

  console.log('*** Actor Tests ***')
  failure |= await actorTests()

  if (failure) {
    console.log('FAILED; setting non-zero exit code')
    process.exitCode = 1
  } else {
    console.log('SUCCESS')
    process.exitCode = 0
  }

  if (process.env.KUBERNETES_MODE === '') {
    console.log('Requesting server shutdown')
    await broadcast('shutdown')
  }

  console.log('Terminating sidecar')
  await shutdown()
}

main()
