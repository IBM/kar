const { actor, call, sys } = require('kar')

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
  const a = actor.proxy('Foo', 123)
  let failure = false

  // ensure clean start (in case test was run previously against this KAR deployment)
  await actor.state.removeAll(a)

  console.log('Testing actor state operations')
  // actor state
  await actor.state.set(a, 'key1', 42)
  await actor.state.set(a, 'key2', 'abc123')
  await actor.state.set(a, 'key3', { field: 'value' })
  await actor.state.set(a, 'key4', null)

  const v1 = await actor.state.get(a, 'key1')
  if (v1 !== 42) {
    console.log(`Failed: get of key1 returned ${v1}`)
    failure = true
  }

  const v2 = await actor.state.get(a, 'key2')
  if (v2 !== 'abc123') {
    console.log(`Failed: get of key2 returned ${v2}`)
    failure = true
  }

  const v3 = await actor.state.getAll(a)
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

  const numNew = await actor.state.setMultiple(a, { key1: 2020, key10: { myData: 1234 } })
  if (numNew !== 1) {
    console.log(`Failed setMultiple: expected 1 new key created but response was ${numNew}`)
    failure = true
  }
  const v3a = await actor.state.getAll(a)
  try {
    if (v3a.key1 !== 2020 ||
    v3a.key2 !== 'abc123' ||
    v3a.key3.field !== 'value' ||
    v3a.key4 != null ||
    v3a.key10.myData !== 1234) {
      console.log(`Failed: getAll ${v3a}`)
      failure = true
    }
  } catch (err) {
    console.log(`Failed during validation of getAll after setMultiple: ${err}.`)
    console.log(`    value was ${v3a}`)
    failure = true
  }

  await actor.state.remove(a, 'key2')
  const v4 = await actor.state.getAll(a)
  if (v4.key2) {
    console.log(`Failed to delete key2: ${v4}`)
    failure = true
  }

  await actor.state.removeAll(a)
  const v5 = await actor.state.getAll(a)
  if (Object.keys(v5).length !== 0) {
    console.log(`Failed to delete all keys: ${v5}`)
    failure = true
  }

  console.log('Testing actor invocation')

  // external synchronous invocation of an actor method
  for (let i = 0; i < 25; i++) {
    const x = await actor.call(actor.proxy('Foo', 'anotherInstance'), 'incrQuiet', i)
    if (x !== i + 1) {
      console.log(`Failed! incr(${i}) returned ${x}`)
      failure = true
    }
  }

  // synchronous invocation via the actor
  const v6 = await actor.call(a, 'incr', 42)
  if (v6 !== 43) {
    console.log(`Failed: unexpected result from incr ${v6}`)
    failure = true
  }

  // asynchronous invocation via the actor
  const v8 = await actor.tell(a, 'incr', 42)
  if (v8 !== 'OK') {
    console.log(`Failed: unexpected result from tell ${v8}`)
    failure = true
  }

  // getter
  const v7 = await actor.call(a, 'field')
  if (v7 !== 42) {
    console.log(`Failed: getter of 'field' returned ${v7}`)
    failure = true
  }

  console.log('Testing actor invocation error handling')
  // error in synchronous invocation
  try {
    console.log(await actor.call(a, 'fail', 'error message 123'))
    console.log('Failed to raise expected error')
    failure = true
  } catch (err) {
    if (verbose) console.log('Caught expected error: ', err.message)
  }

  // undefined method
  try {
    console.log(await actor.call(a, 'missing', 'error message 123'))
    console.log('Failed. No error raised invoking missing method')
    failure = true
  } catch (err) {
    if (verbose) console.log('Caught expected error: ', err.message)
  }

  // reentrancy
  const v9 = await actor.call(a, 'reenter', 42)
  if (v9 !== 43) {
    console.log(`Failed: unexpected result from reenter ${v9}`)
    failure = true
  }

  return failure
}

async function pubSubTests () {
  let failure = false

  const v = await call('myService', 'pubsub', 'topic1')
  if (v !== 'OK') {
    console.log('Failed: pubsub')
    failure = true
  }

  return failure
}

async function testTermination (failure) {
  if (failure) {
    console.log('FAILED; setting non-zero exit code')
    process.exitCode = 1
  } else {
    console.log('SUCCESS')
    process.exitCode = 0
  }

  if (!truthy(process.env.KUBERNETES_MODE)) {
    console.log('Requesting server shutdown')
    await sys.broadcast('shutdown')
  }

  console.log('Terminating sidecar')
  await sys.shutdown()
}

async function main () {
  var failure = false

  console.log('*** Service Tests ***')
  failure |= await serviceTests()

  console.log('*** Actor Tests ***')
  failure |= await actorTests()

  console.log('*** PubSub Tests ***')
  failure |= await pubSubTests()

  testTermination(failure)
}

main()
