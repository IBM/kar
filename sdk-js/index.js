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

const { triggerAsyncId } = require('async_hooks')
const express = require('express')
const http2 = require('http2')
const morgan = require('morgan') // for logging http requests and responses
const spdy = require('spdy')

if (!process.env.KAR_RUNTIME_PORT) {
  console.error('KAR_RUNTIME_PORT must be set. Aborting.')
  process.exit(1)
}

const session = http2.connect(`http://localhost:${process.env.KAR_RUNTIME_PORT}`)

// assumes utf8
function rawFetch (path, { method, headers, body } = {}) {
  const obj = { ':path': path }
  if (method) obj[':method'] = method
  Object.assign(obj, headers)
  return new Promise((resolve, reject) => {
    const req = session.request(obj)
    req.setEncoding('utf8')
    if (Number(process.env.KAR_REQUEST_TIMEOUT) >= 0) req.setTimeout(Number(process.env.KAR_REQUEST_TIMEOUT))
    let text = ''
    const res = { ok: true, text: () => Promise.resolve(text) }
    req.on('response', headers => {
      res.headers = headers
      res.status = headers[':status']
      res.ok = headers[':status'] >= 200 && headers[':status'] < 300
    })
    req.on('data', s => { text += s })
    req.on('error', reject)
    req.on('end', () => resolve(res))
    if (body) req.write(body, 'utf8')
    req.end()
  })
}

// retry http requests up to 10 times on failure or 503 error
const fetch = require('fetch-retry')(rawFetch, { retries: 10, retryOn: [503] })

// url prefix for http requests to sidecar
const url = '/kar/v1/'

// parse http response
const parse = res => res.text().then(text => { // parse to string first
  if (!res.ok) throw new Error(text) // if error response return error string
  try { // try parsing to json object
    return text.length > 0 ? JSON.parse(text) : undefined // return undefined if empty string otherwise parse json object
  } catch (err) {
    return text // return string if not json
  }
})

// parse actor response
const parseActor = res => res.text().then(text => { // parse to string first
  if (!res.ok) throw new Error(text) // if error response return error string
  if (res.status === 204) return undefined
  if (res.status === 202) return text
  let obj
  try { // try parsing to json object
    obj = JSON.parse(text)
  } catch (err) {
    throw new Error(text)
  }
  if (obj.error) {
    const err = new Error(obj.message)
    err.stack = obj.stack
    throw err
  } else {
    return obj.value
  }
})

// http post: json stringify request body, parse response body
function post (api, body, headers) {
  return fetch(url + api, { method: 'POST', body: JSON.stringify(body), headers }).then(parse)
}

// http post: json stringify request body, parse response body
function postActor (api, body, headers) {
  return fetch(url + api, { method: 'POST', body: JSON.stringify(body), headers }).then(parseActor)
}

// http put: json stringify request body, parse response body
function put (api, body, headers) {
  return fetch(url + api, { method: 'PUT', body: JSON.stringify(body), headers }).then(parse)
}

// http get: parse response body
function get (api) {
  return fetch(url + api).then(parse)
}

// http head: return response headers
function head (api) {
  return fetch(url + api, { method: 'HEAD' }).then(res => res.headers)
}

// http del: parse response body
function del (api) {
  return fetch(url + api, { method: 'DELETE' }).then(parse)
}

// check if string value is truthy
const truthy = s => s && s.toLowerCase() !== 'false' && s !== '0'

/***************************************************
 * public methods intended for application programming
 * API Documentation is located in index.d.ts
 ***************************************************/

const invoke = (service, path, options) => fetch(url + `service/${service}/call/${path}`, options)

const tell = (service, path, body) => post(`service/${service}/call/${path}`, body, { 'Content-Type': 'application/json', Pragma: 'async' })

const call = (service, path, body) => post(`service/${service}/call/${path}`, body, { 'Content-Type': 'application/json' })

function encodeTailCall (service, path, payload) {
  return { tailCall: true, value: { serviceName: service, path: '/' + path, payload, method: 'POST' } }
}

const resolver = request => () => fetch(url + 'await', { method: 'POST', body: request, headers: { 'Content-Type': 'text/plain' } }).then(parse)

const resolverActor = request => () => fetch(url + 'await', { method: 'POST', body: request, headers: { 'Content-Type': 'text/plain' } }).then(parseActor)

function asyncCall (service, path, body) {
  return post(`service/${service}/call/${path}`, body, { 'Content-Type': 'application/json', Pragma: 'promise' }).then(resolver)
}

function actorProxy (type, id) { return { kar: { type, id } } }

const actorTell = (actor, path, ...args) => post(`actor/${actor.kar.type}/${actor.kar.id}/call/${path}`, args, { 'Content-Type': 'application/kar+json', Pragma: 'async' })

function actorCall (...args) {
  if (typeof args[1] === 'string') {
    // call (callee:Actor, path:string, ...args:any[]):Promise<any>;
    // TODO: Once we release an SDK with rootCall, then remove this option
    const ta = args.shift()
    const path = args.shift()
    return postActor(`actor/${ta.kar.type}/${ta.kar.id}/call/${path}`, args, { 'Content-Type': 'application/kar+json' })
  } else {
    //  call (from:Actor, callee:Actor, path:string, ...args:any[]):Promise<any>;
    const sa = args.shift()
    const ta = args.shift()
    const path = args.shift()
    return postActor(`actor/${ta.kar.type}/${ta.kar.id}/call/${path}?session=${sa.kar.session}`, args, { 'Content-Type': 'application/kar+json' })
  }
}

function actorRootCall (...args) {
  // call (callee:Actor, path:string, ...args:any[]):Promise<any>;
  const ta = args.shift()
  const path = args.shift()
  return postActor(`actor/${ta.kar.type}/${ta.kar.id}/call/${path}`, args, { 'Content-Type': 'application/kar+json' })
}

function actorAsyncCall (...args) {
  if (typeof args[1] === 'string') {
    // call (callee:Actor, path:string, ...args:any[]):Promise<any>;
    const ta = args.shift()
    const path = args.shift()
    return postActor(`actor/${ta.kar.type}/${ta.kar.id}/call/${path}`, args, { 'Content-Type': 'application/kar+json', Pragma: 'promise' }).then(resolverActor)
  } else {
    //  call (from:Actor, callee:Actor, path:string, ...args:any[]):Promise<any>;
    const sa = args.shift()
    const ta = args.shift()
    const path = args.shift()
    return postActor(`actor/${ta.kar.type}/${ta.kar.id}/call/${path}?session=${sa.kar.session}`, args, { 'Content-Type': 'application/kar+json', Pragma: 'promise' }).then(resolverActor)
  }
}

function actorEncodeTailCall (...args) {
  const ta = args.shift()
  const path = args.shift()
  const payload = JSON.stringify(args)
  return { tailCall: true, value: { actorType: ta.kar.type, actorId: ta.kar.id, path: '/' + path, payload } }
}

function actorEncodeTailCallReleasingLock (...args) {
  const ta = args.shift()
  const path = args.shift()
  const payload = JSON.stringify(args)
  return { tailCall: true, value: { actorType: ta.kar.type, actorId: ta.kar.id, releaseLock: 'true', path: '/' + path, payload } }
}

const actorDelete = (actor) => del(`actor/${actor.kar.type}/${actor.kar.id}`)

const actorCancelReminder = (actor, reminderId) => reminderId ? del(`actor/${actor.kar.type}/${actor.kar.id}/reminders/${reminderId}?nilOnAbsent=true`) : del(`actor/${actor.kar.type}/${actor.kar.id}/reminders`)

const actorGetReminder = (actor, reminderId) => reminderId ? get(`actor/${actor.kar.type}/${actor.kar.id}/reminders/${reminderId}?nilOnAbsent=true`) : get(`actor/${actor.kar.type}/${actor.kar.id}/reminders`)

function actorScheduleReminder (actor, path, options, ...args) {
  const opts = { path: `/${path}`, targetTime: options.targetTime, period: options.period, data: args }
  return put(`actor/${actor.kar.type}/${actor.kar.id}/reminders/${options.id}`, opts)
}

const actorStateGet = (actor, key) => get(`actor/${actor.kar.type}/${actor.kar.id}/state/${key}?nilOnAbsent=true`)

const actorStateGetAll = (actor) => get(`actor/${actor.kar.type}/${actor.kar.id}/state`)

const actorStateContains = (actor, key) => head(`actor/${actor.kar.type}/${actor.kar.id}/state/${key}`).status === 200

const actorStateSet = (actor, key, value = {}) => put(`actor/${actor.kar.type}/${actor.kar.id}/state/${key}`, value)

const actorStateSetMultiple = (actor, state = {}) => post(`actor/${actor.kar.type}/${actor.kar.id}/state`, { updates: state }).then(res => res.added)

const actorStateRemove = (actor, key) => del(`actor/${actor.kar.type}/${actor.kar.id}/state/${key}`)

const actorStateRemoveSome = (actor, keys = []) => post(`actor/${actor.kar.type}/${actor.kar.id}/state`, { removals: keys }).then(res => res.removed)

const actorStateRemoveAll = (actor) => del(`actor/${actor.kar.type}/${actor.kar.id}/state`)

const actorStateUpdate = (actor, changes) => post(`actor/${actor.kar.type}/${actor.kar.id}/state`, changes)

const actorSubmapGet = (actor, key, subkey) => get(`actor/${actor.kar.type}/${actor.kar.id}/state/${key}/${subkey}?nilOnAbsent=true`)

const actorSubmapGetAll = (actor, key) => post(`actor/${actor.kar.type}/${actor.kar.id}/state/${key}`, { op: 'get' })

const actorSubmapContains = (actor, key, subkey) => head(`actor/${actor.kar.type}/${actor.kar.id}/state/${key}/${subkey}`).then(headers => headers[':status'] === 200)

const actorSubmapSet = (actor, key, subkey, value = {}) => put(`actor/${actor.kar.type}/${actor.kar.id}/state/${key}/${subkey}`, value)

const actorSubmapSetMultiple = (actor, key, state = {}) => post(`actor/${actor.kar.type}/${actor.kar.id}/state`, { submapupdates: { [key]: state } }).then(res => res.added)

const actorSubmapRemove = (actor, key, subkey) => del(`actor/${actor.kar.type}/${actor.kar.id}/state/${key}/${subkey}`)

const actorSubmapRemoveSome = (actor, key, keys = []) => post(`actor/${actor.kar.type}/${actor.kar.id}/state`, { submapremovals: { [key]: keys } }).then(res => res.removed)

const actorSubmapRemoveAll = (actor, key) => post(`actor/${actor.kar.type}/${actor.kar.id}/state/${key}`, { op: 'clear' })

const actorSubmapKeys = (actor, key) => post(`actor/${actor.kar.type}/${actor.kar.id}/state/${key}`, { op: 'keys' })

const actorSubmapSize = (actor, key) => post(`actor/${actor.kar.type}/${actor.kar.id}/state/${key}`, { op: 'size' })

const shutdown = () => post('system/shutdown').then(_ => session.close())

const eventsCancelSubscription = (actor, id) => id ? del(`actor/${actor.kar.type}/${actor.kar.id}/events/${id}`) : del(`actor/${actor.kar.type}/${actor.kar.id}/events`)

const eventsGetSubscription = (actor, id) => id ? get(`actor/${actor.kar.type}/${actor.kar.id}/events/${id}`) : get(`actor/${actor.kar.type}/${actor.kar.id}/events`)

function eventsCreateSubscription (actor, path, topic, options = {}) {
  const id = options.id || topic
  const contentType = options.contentType
  return put(`actor/${actor.kar.type}/${actor.kar.id}/events/${id}`, { path: `/${path}`, topic, contentType })
}

const eventsCreateTopic = (topic, options) => put(`event/${topic}`, options)

const eventsDeleteTopic = (topic) => del(`event/${topic}`)

const eventsPublish = (topic, event) => post(`event/${topic}/publish`, event)

const systemGet = (query) => fetch(url + 'system/information/' + query, { headers: { Accept: 'application/json' } }).then(parse)

/***************************************************
 * End of public methods intended for application programming
 **************************************************/

const errorHandler = [
  (err, req, res, next) => Promise.resolve()
    .then(_ => {
      err.stack += `\n    at <kar> ${req.originalUrl}` // add request url to stack trace
      const body = { error: true } // sanitize error object
      body.message = typeof err.message === 'string' ? err.message : typeof err === 'string' ? err : 'Internal Server Error'
      body.stack = typeof err.stack === 'string' ? err.stack : new Error(body.message).stack
      return res.status(200).type('application/kar+json').send(body) // return error
    })
    .catch(next)] // forward errors to next middleware (but there should not be any...)

// h2c protocol wrapper
const h2c = app => spdy.createServer({ spdy: { plain: true, ssl: false, connection: { maxStreams: 262144 } } }, app).setTimeout(0)
/***************************************************
 * Start of Actor runtime implementation
 ***************************************************/

const table = {} // live actor instances: table[actorType][actorId]

function actorRuntime (actors) {
  const router = express.Router()

  if (truthy(process.env.KAR_VERBOSE)) router.use([morgan('--> :date[iso] :method :url', { immediate: true }), morgan('<-- :date[iso] :method :url :status - :response-time ms')])

  router.use(express.json({ type: '*/*' })) // unconditionally parse request bodies to json

  // actor activation route
  router.get('/kar/impl/v1/actor/:type/:id', (req, res, next) => {
    const Actor = actors[req.params.type]
    if (Actor == null) return res.status(404).type('text/plain').send(`no actor type ${req.params.type}`)
    if (table[req.params.type] && table[req.params.type][req.params.id]) {
      return res.status(200).type('text/plain').send('existing instance')
    }
    return Promise.resolve()
      .then(_ => {
        table[req.params.type] = table[req.params.type] || {}
        const actor = new Actor(req.params.id)
        table[req.params.type][req.params.id] = actor
        table[req.params.type][req.params.id].kar = { type: req.params.type, id: req.params.id, session: req.query.session }
      }) // instantiate actor and add to index
      .then(_ => { // run optional activate callback
        if (typeof table[req.params.type][req.params.id].activate === 'function') return table[req.params.type][req.params.id].activate()
      })
      .then(_ => {
        table[req.params.type][req.params.id].kar.session = undefined
        return res.sendStatus(201) // Created
      })
      .catch(next)
  })

  // actor deactivation route
  router.delete('/kar/impl/v1/actor/:type/:id', (req, res, next) => {
    const Actor = actors[req.params.type]
    if (Actor == null) return res.status(404).type('text/plain').send(`no actor type ${req.params.type}`)
    const actor = (table[req.params.type] || {})[req.params.id]
    if (actor == null) return res.status(404).type('text/plain').send(`no actor with type ${req.params.type} and id ${req.params.id}`)
    return Promise.resolve()
      .then(_ => { // run optional deactivate callback
        delete actor.kar.session
        if (typeof actor.deactivate === 'function') return actor.deactivate()
      })
      .then(_ => delete table[req.params.type][req.params.id]) // remove actor from index
      .then(_ => res.sendStatus(200)) // OK
      .catch(next)
  })

  // method invocation route
  router.post('/kar/impl/v1/actor/:type/:id/:method', (req, res, next) => {
    const Actor = actors[req.params.type]
    if (Actor == null) return res.status(404).type('text/plain').send(`no actor type ${req.params.type}`)
    const actor = (table[req.params.type] || {})[req.params.id]
    if (actor == null) return res.status(404).type('text/plain').send(`no actor with type ${req.params.type} and id ${req.params.id}`)
    if (!(req.params.method in actor)) return res.status(404).type('text/plain').send(`no ${req.params.method} in actor with type ${req.params.type} and id ${req.params.id}`)
    const priorSession = actor.kar.session
    return Promise.resolve()
      .then(_ => {
        actor.kar.session = req.query.session
        if (typeof actor[req.params.method] === 'function') return actor[req.params.method](...req.body)
        return actor[req.params.method]
      }) // invoke method on actor
      .then(value => {
        actor.kar.session = priorSession
        if (value === undefined) {
          return res.status(204).send()
        } else {
          if (value.tailCall) {
            return res.status(200).type('application/kar+json').send(value)
          } else {
            return res.status(200).type('application/kar+json').send({ value })
          }
        }
      })
      .catch(_ => {
        actor.kar.session = priorSession
        next()
      })
  })

  // actor type validation route
  router.head('/kar/impl/v1/actor/:type', (req, res, next) => {
    const Actor = actors[req.params.type]
    if (Actor == null) return res.status(404).send()
    return res.status(200).send()
  })

  // health route
  router.get('/kar/impl/v1/system/health', (req, res, next) => {
    return res.status(200).type('text/plain').send('Peachy Keen!')
  })

  router.use(errorHandler)

  return router
}

/***************************************************
 * End of Actor runtime implementation
 ***************************************************/

// exports
module.exports = {
  invoke,
  tell,
  call,
  asyncCall,
  tailCall: encodeTailCall,
  actor: {
    proxy: actorProxy,
    tell: actorTell,
    call: actorCall,
    rootCall: actorRootCall,
    asyncCall: actorAsyncCall,
    remove: actorDelete,
    tailCall: actorEncodeTailCall,
    tailCallReleaasingLock: actorEncodeTailCallReleasingLock,
    reminders: {
      cancel: actorCancelReminder,
      get: actorGetReminder,
      schedule: actorScheduleReminder
    },
    state: {
      get: actorStateGet,
      getAll: actorStateGetAll,
      contains: actorStateContains,
      set: actorStateSet,
      setMultiple: actorStateSetMultiple,
      remove: actorStateRemove,
      removeSome: actorStateRemoveSome,
      removeAll: actorStateRemoveAll,
      update: actorStateUpdate,
      submap: {
        get: actorSubmapGet,
        getAll: actorSubmapGetAll,
        contains: actorSubmapContains,
        set: actorSubmapSet,
        setMultiple: actorSubmapSetMultiple,
        remove: actorSubmapRemove,
        removeSome: actorSubmapRemoveSome,
        removeAll: actorSubmapRemoveAll,
        keys: actorSubmapKeys,
        size: actorSubmapSize
      }
    }
  },
  events: {
    cancelSubscription: eventsCancelSubscription,
    getSubscription: eventsGetSubscription,
    subscribe: eventsCreateSubscription,
    createTopic: eventsCreateTopic,
    deleteTopic: eventsDeleteTopic,
    publish: eventsPublish
  },
  sys: {
    actorRuntime,
    get: systemGet,
    shutdown,
    h2c,
    errorHandler
  }
}
