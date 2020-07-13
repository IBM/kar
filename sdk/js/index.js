const express = require('express')
const http2 = require('http2')
const morgan = require('morgan') // for logging http requests and responses
const spdy = require('spdy')

const agent = http2.connect(`http://localhost:${process.env.KAR_RUNTIME_PORT || 3500}`)

function rawFetch (path, options) {
  const obj = { ':path': path }
  if (options.method) {
    obj[':method'] = options.method
  }
  Object.assign(obj, options.headers)
  return new Promise((resolve, reject) => {
    const req = options.agent.request(obj)
    req.setEncoding('utf8')
    if (Number(process.env.KAR_REQUEST_TIMEOUT) >= 0) {
      req.setTimeout(Number(process.env.KAR_REQUEST_TIMEOUT))
    }
    let ok = true
    let text = ''
    req.on('response', headers => { ok = headers[':status'] < 300 })
    req.on('data', s => { text += s })
    req.on('error', reject)
    req.on('end', () => resolve({ ok, text: () => Promise.resolve(text) }))
    if (options.body) req.write(options.body, 'utf8')
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
  try { // try parsing to json object
    const obj = JSON.parse(text)
    if (obj.error) {
      const err = new Error(obj.message)
      err.stack = obj.stack
      throw err
    } else {
      return obj.value
    }
  } catch (err) {
    throw new Error(text)
  }
})

// http post: json stringify request body, parse response body
function post (api, body, headers) {
  return fetch(url + api, { method: 'POST', body: JSON.stringify(body), headers, agent }).then(parse)
}

// http post: json stringify request body, parse response body
function postActor (api, body, headers) {
  return fetch(url + api, { method: 'POST', body: JSON.stringify(body), headers, agent }).then(parseActor)
}

// http put: json stringify request body, parse response body
function put (api, body, headers) {
  return fetch(url + api, { method: 'PUT', body: JSON.stringify(body), headers, agent }).then(parse)
}

// http get: parse response body
function get (api) {
  return fetch(url + api, { agent }).then(parse)
}

// http del: parse response body
function del (api) {
  return fetch(url + api, { method: 'DELETE', agent }).then(parse)
}

// check if string value is truthy
const truthy = s => s && s.toLowerCase() !== 'false' && s !== '0'

/***************************************************
 * public methods intended for application programming
 * API Documentation is located in index.d.ts
 ***************************************************/

const tell = (service, path, body) => post(`service/${service}/call/${path}`, body, { 'Content-Type': 'application/json', Pragma: 'async' })

const call = (service, path, body) => post(`service/${service}/call/${path}`, body, { 'Content-Type': 'application/json' })

const resolver = request => () => fetch(url + 'await', { method: 'POST', body: request, headers: { 'Content-Type': 'text/plain' }, agent }).then(parse)

const resolverActor = request => () => fetch(url + 'await', { method: 'POST', body: request, headers: { 'Content-Type': 'text/plain' }, agent }).then(parseActor)

function asyncCall (service, path, body) {
  return post(`service/${service}/call/${path}`, body, { 'Content-Type': 'application/json', Pragma: 'promise' }).then(resolver)
}

function actorProxy (type, id) { return { kar: { type, id } } }

const actorTell = (actor, path, ...args) => post(`actor/${actor.kar.type}/${actor.kar.id}/call/${path}`, args, { 'Content-Type': 'application/kar+json', Pragma: 'async' })

function actorCall (...args) {
  if (typeof args[1] === 'string') {
    // call (callee:Actor, path:string, ...args:any[]):Promise<any>;
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

const actorCancelReminder = (actor, reminderId) => reminderId ? del(`actor/${actor.kar.type}/${actor.kar.id}/reminders/${reminderId}?nilOnAbsent=true`) : del(`actor/${actor.kar.type}/${actor.kar.id}/reminders`)

const actorGetReminder = (actor, reminderId) => reminderId ? get(`actor/${actor.kar.type}/${actor.kar.id}/reminders/${reminderId}?nilOnAbsent=true`) : get(`actor/${actor.kar.type}/${actor.kar.id}/reminders`)

function actorScheduleReminder (actor, path, options, ...args) {
  const opts = { path: `/${path}`, targetTime: options.targetTime, period: options.period, data: args }
  return post(`actor/${actor.kar.type}/${actor.kar.id}/reminders/${options.id}`, opts)
}

const actorGetState = (actor, key) => get(`actor/${actor.kar.type}/${actor.kar.id}/state/${key}?nilOnAbsent=true`)

const actorSetState = (actor, key, value = {}) => put(`actor/${actor.kar.type}/${actor.kar.id}/state/${key}`, value)

const actorRemoveState = (actor, key) => del(`actor/${actor.kar.type}/${actor.kar.id}/state/${key}`)

const actorGetAllState = (actor) => get(`actor/${actor.kar.type}/${actor.kar.id}/state`)

const actorSetStateMultiple = (actor, state = {}) => post(`actor/${actor.kar.type}/${actor.kar.id}/state`, state)

const actorRemoveAllState = (actor) => del(`actor/${actor.kar.type}/${actor.kar.id}/state`)

const shutdown = () => post('system/shutdown').then(() => agent.close())

function publish (topic, event) {
  // Ensure event is of the correct type.
  if (typeof event !== 'object') throw new Error('publish: event must be of type "object"')
  if (typeof event.spec !== 'object') throw new Error('publish: event.spec must be of type "object"')
  if (typeof event.spec.payload !== 'object') throw new Error('publish: event.spec.payload must be of type "object"')

  // Construct POST request from input Cloudevent.
  const payload = event.spec.payload

  // Check mandatory fields for a Cloudevent are present.
  if (typeof topic === 'undefined') throw new Error('publish: must define "topic"')
  if (typeof payload.id === 'undefined') throw new Error('publish: must define "id"')
  if (typeof payload.source === 'undefined') throw new Error('publish: must define "source"')
  if (typeof payload.type === 'undefined') throw new Error('publish: must define "type"')

  // Set specversion if not set.
  if (typeof payload.specversion === 'undefined') payload.specversion = '1.0'

  return post(`event/${topic}/publish`, payload)
}

const subscribe = (topic, path, opts) => post(`event/${topic}/subscribe`, Object.assign({ path: `/${path}` }, opts))

const unsubscribe = (topic, opts) => post(`event/${topic}/unsubscribe`, opts)

function actorSubscribe (actor, topic, path, params = {}) {
  const id = params.id || topic
  return post(`actor/${actor.kar.type}/${actor.kar.id}/events/${id}`, Object.assign({ path: `/${path}`, topic }, params))
}

function actorUnsubscribe (actor, topic, params = {}) {
  const id = params.id || topic
  return del(`actor/${actor.kar.type}/${actor.kar.id}/events/${id}`, Object.assign({ topic }, params))
}

/***************************************************
 * End of public methods intended for application programming
 **************************************************/

const errorHandler = [
  (err, req, res, next) => Promise.resolve()
    .then(() => {
      err.stack += `\n    at <kar> ${req.originalUrl}` // add request url to stack trace
      const body = { error: true } // sanitize error object
      body.message = typeof err.message === 'string' ? err.message : typeof err === 'string' ? err : 'Internal Server Error'
      body.stack = typeof err.stack === 'string' ? err.stack : new Error(body.message).stack
      return res.json(body) // return error
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
    if (Actor == null) return res.status(404).send(`no actor type ${req.params.type}`)
    if (table[req.params.type] && table[req.params.type][req.params.id]) {
      return res.status(200).send('existing instance')
    }
    return Promise.resolve()
      .then(_ => {
        table[req.params.type] = table[req.params.type] || {}
        const actor = new Actor(req.params.id)
        table[req.params.type][req.params.id] = actor
        table[req.params.type][req.params.id].kar = { type: req.params.type, id: req.params.id }
      }) // instantiate actor and add to index
      .then(_ => { // run optional activate callback
        if (typeof table[req.params.type][req.params.id].activate === 'function') return table[req.params.type][req.params.id].activate()
      })
      .then(_ => res.sendStatus(201)) // Created
      .catch(next)
  })

  // actor deactivation route
  router.delete('/kar/impl/v1/actor/:type/:id', (req, res, next) => {
    const Actor = actors[req.params.type]
    if (Actor == null) return res.status(404).send(`no actor type ${req.params.type}`)
    const actor = table[req.params.type][req.params.id]
    if (actor == null) return res.status(404).send(`no actor with type ${req.params.type} and id ${req.params.id}`)
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
  router.post('/kar/impl/v1/actor/:type/:id/:session/:method', (req, res, next) => {
    const Actor = actors[req.params.type]
    if (Actor == null) return res.status(404).send(`no actor type ${req.params.type}`)
    const actor = table[req.params.type][req.params.id]
    if (actor == null) return res.status(404).send(`no actor with type ${req.params.type} and id ${req.params.id}`)
    return Promise.resolve()
      .then(_ => {
        if (req.params.method in actor) {
          // NOTE: session intentionally not cleared before return (could be nested call in same session)
          actor.kar.session = req.params.session
          if (typeof actor[req.params.method] === 'function') {
            return actor[req.params.method](...req.body)
          }
          return actor[req.params.method]
        }
        throw new Error(`${req.params.method} is not defined on actor with type ${req.params.type} and id ${req.params.id}`)
      }) // invoke method on actor
      .then(value => res.json({ value })) // stringify invocation result
      .catch(next)
  }) // delegate error handling to postprocessor

  router.use(errorHandler)

  return router
}

/***************************************************
 * End of Actor runtime implementation
 ***************************************************/

// exports
module.exports = {
  tell,
  call,
  asyncCall,
  publish,
  subscribe,
  unsubscribe,
  actor: {
    proxy: actorProxy,
    tell: actorTell,
    call: actorCall,
    asyncCall: actorAsyncCall,
    subscribe: actorSubscribe,
    unsubscribe: actorUnsubscribe,
    reminders: {
      cancel: actorCancelReminder,
      get: actorGetReminder,
      schedule: actorScheduleReminder
    },
    state: {
      get: actorGetState,
      set: actorSetState,
      setMultiple: actorSetStateMultiple,
      remove: actorRemoveState,
      getAll: actorGetAllState,
      removeAll: actorRemoveAllState
    }
  },
  sys: {
    actorRuntime,
    shutdown,
    h2c,
    errorHandler
  }
}
