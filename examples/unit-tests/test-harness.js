/*
 * Copyright IBM Corporation 2020,2022
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

const { actor, call, events, sys } = require('kar-sdk')

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
      console.log(`Failed: getAll ${JSON.stringify(v3)}`)
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
      console.log(`Failed: getAll ${JSON.stringify(v3a)}`)
      failure = true
    }
  } catch (err) {
    console.log(`Failed during validation of getAll after setMultiple: ${err}.`)
    console.log(`    value was ${v3a}`)
    failure = true
  }
  const numRemoved = await actor.state.removeSome(a, ['key1', 'keyNotHere', 'key10'])
  if (numRemoved !== 2) {
    console.log(`removeSome removed ${numRemoved} keys, was expecting to remove 2`)
    failure = true
  }
  await actor.state.submap.set(a, 'famous', 'Allen', 'Fran')
  await actor.state.submap.setMultiple(a, 'famous', { Turing: 'Alan', Knuth: 'Don' })
  const fa = await actor.state.submap.get(a, 'famous', 'Allen')
  const dk = await actor.state.submap.get(a, 'famous', 'Knuth')
  if (fa !== 'Fran' || dk !== 'Don') {
    console.log(`Failed to look up famous people: ${fa} or ${dk}`)
    failure = true
  }
  const fae = await actor.state.submap.contains(a, 'famous', 'Allen')
  if (fae !== true) {
    console.log('did not find contained key famous/Allen')
    failure = true
  }
  const fjd = await actor.state.submap.contains(a, 'famous', 'Doe')
  if (fjd !== false) {
    console.log('found non-contained key famous/Doe')
    failure = true
  }
  const nf = await actor.state.submap.size(a, 'famous')
  if (nf !== 3) {
    console.log(`Unexpected number of famous people: ${nf}`)
    failure = true
  }
  const fpm = await actor.state.submap.getAll(a, 'famous')
  if (fpm.Allen !== 'Fran' || fpm.Turing !== 'Alan' || fpm.Knuth !== 'Don') {
    console.log('Missing famous person entry from getSubmaps')
    failure = true
  }
  await actor.state.submap.remove(a, 'famous', 'Knuth')
  const dk2 = await actor.state.submap.get(a, 'famous', 'Knuth')
  if (dk2) {
    console.log('Failed to remove Don Knuth from famous map')
    failure = true
  }
  const fp = await actor.state.submap.keys(a, 'famous')
  if (!fp.includes('Allen') || !fp.includes('Turing')) {
    console.log(`Unexpected set of famous people ${fp}`)
    failure = true
  }
  const numRemovedSub = await actor.state.submap.removeSome(a, 'famous', ['Allen', 'Knuth'])
  if (numRemovedSub !== 1) {
    console.log(`Expected removeSomeSubmap to remove 1; actually removed ${numRemovedSub}`)
    failure = true
  }
  await actor.state.submap.removeAll(a, 'famous')
  if (await actor.state.submap.size(a, 'famous') !== 0) {
    console.log('Submap clear did not remove all keys')
    failure = true
  }
  const fpm2 = await actor.state.submap.getAll(a, 'famous')
  if (!fpm2.size === 0) {
    console.log('Submap get on empty subMap did not return empty map')
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
    console.log(`Failed to delete all keys: ${JSON.stringify(v5)}`)
    failure = true
  }

  console.log('Testing actor invocation')

  // external synchronous invocation of an actor method
  for (let i = 0; i < 25; i++) {
    const x = await actor.rootCall(actor.proxy('Foo', 'anotherInstance'), 'incrQuiet', i)
    if (x !== i + 1) {
      console.log(`Failed! incr(${i}) returned ${x}`)
      failure = true
    }
  }

  // synchronous invocation via the actor
  const v6 = await actor.rootCall(a, 'incr', 42)
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
  const v7 = await actor.rootCall(a, 'field')
  if (v7 !== 42) {
    console.log(`Failed: getter of 'field' returned ${v7}`)
    failure = true
  }

  console.log('Testing actor invocation error handling')
  // error in synchronous invocation
  try {
    console.log(await actor.rootCall(a, 'fail', 'error message 123'))
    console.log('Failed to raise expected error')
    failure = true
  } catch (err) {
    if (verbose) console.log('Caught expected error: ', err.message)
  }

  // undefined method
  try {
    console.log(await actor.rootCall(a, 'missing', 'error message 123'))
    console.log('Failed. No error raised invoking missing method')
    failure = true
  } catch (err) {
    if (verbose) console.log('Caught expected error: ', err.message)
  }

  // reentrancy
  const v9 = await actor.rootCall(a, 'reenter', 42)
  if (v9 !== 43) {
    console.log(`Failed: unexpected result from reenter ${v9}`)
    failure = true
  }

  // asyncCall invocation via the actor
  const v10 = await actor.asyncCall(a, 'incr', 42)
  const v11 = await v10()
  if (v11 !== 43) {
    console.log(`Failed: unexpected result from asyncCall ${v11}`)
    failure = true
  }

  return failure
}

async function pubSubTests () {
  const a = actor.proxy('Foo', 456)
  let failure = false
  const topic = 'test-topic2'

  await events.createTopic(topic)

  const v1 = await actor.rootCall(a, 'pubsub', topic)
  if (v1 !== 'OK') {
    console.log('Failed: pubsub')
    failure = true
  }

  let i
  for (i = 30; i > 0; i--) { // poll
    const v2 = await actor.rootCall(a, 'check', topic)
    if (v2 === true) break
    await new Promise(resolve => setTimeout(resolve, 500)) // wait
  }
  if (i === 0) {
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

  console.log('Terminating sidecar')
  await sys.shutdown()
}

async function main () {
  let failure = false

  console.log('*** Service Tests ***')
  failure |= await serviceTests()

  console.log('*** Actor Tests ***')
  failure |= await actorTests()

  console.log('*** PubSub Tests ***')
  failure |= await pubSubTests()

  testTermination(failure)
}

main()
