const express = require('express')
const http2 = require('http2')
const parser = require('body-parser') // for parsing http requests
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

// headers for http requests to sidecar
const headers = { 'Content-Type': 'application/json' }

// parse http response
const parse = res => res.text().then(text => { // parse to string first
  if (!res.ok) throw new Error(text) // if error response return error string
  try { // try parsing to json object
    return text.length > 0 ? JSON.parse(text) : undefined // return undefined if empty string otherwise parse json object
  } catch (err) {
    return text // return string if not json
  }
})

// http post: json stringify request body, parse response body
function post (api, body) {
  return fetch(url + api, { method: 'POST', body: JSON.stringify(body), headers, agent }).then(parse)
}

// http put: json stringify request body, parse response body
function put (api, body) {
  return fetch(url + api, { method: 'PUT', body: JSON.stringify(body), headers, agent }).then(parse)
}

// http get: parse response body
function get (api) {
  return fetch(url + api, { headers, agent }).then(parse)
}

// http del: parse response body
function del (api) {
  return fetch(url + api, { method: 'DELETE', headers, agent }).then(parse)
}

// check if string value is truthy
const truthy = s => s && s.toLowerCase() !== 'false' && s !== '0'

/***************************************************
 * public methods intended for application programming
 * API Documentation is located in index.d.ts
 ***************************************************/

const tell = (service, path, body) => post(`service/${service}/tell/${path}`, body)

const call = (service, path, body) => post(`service/${service}/call/${path}`, body)

function actorProxy (type, id) { return { kar: { type, id } } }

const actorTell = (actor, path, ...args) => post(`actor/${actor.kar.type}/${actor.kar.id}/tell/${path}`, args.length === 0 ? undefined : args)

function actorCall (...args) {
  if (typeof args[1] === 'string') {
    // call (callee:Actor, path:string, ...args:any[]):Promise<any>;
    const ta = args.shift()
    const path = args.shift()
    const rest = args.length === 0 ? undefined : args
    return post(`actor/${ta.kar.type}/${ta.kar.id}/call/${path}`, rest)
  } else {
    //  call (from:Actor, callee:Actor, path:string, ...args:any[]):Promise<any>;
    const sa = args.shift()
    const ta = args.shift()
    const path = args.shift()
    const rest = args.length === 0 ? undefined : args
    return post(`actor/${ta.kar.type}/${ta.kar.id}/call/${path}?session=${sa.kar.session}`, rest)
  }
}

const actorCancelReminder = (actor, reminderId) => reminderId ? del(`actor/${actor.kar.type}/${actor.kar.id}/reminders/${reminderId}?nilOnAbsent=true`) : del(`actor/${actor.kar.type}/${actor.kar.id}/reminders`)

const actorGetReminder = (actor, reminderId) => reminderId ? get(`actor/${actor.kar.type}/${actor.kar.id}/reminders/${reminderId}?nilOnAbsent=true`) : get(`actor/${actor.kar.type}/${actor.kar.id}/reminders`)

function actorScheduleReminder (actor, path, id, targetTime, period, ...args) {
  const opts = { id, period, path: `/${path}`, targetTime, data: args.length === 0 ? undefined : args }
  return post(`actor/${actor.kar.type}/${actor.kar.id}/reminders`, opts)
}

const actorGetState = (actor, key) => get(`actor/${actor.kar.type}/${actor.kar.id}/state/${key}?nilOnAbsent=true`)

const actorSetState = (actor, key, value = {}) => put(`actor/${actor.kar.type}/${actor.kar.id}/state/${key}`, value)

const actorRemoveState = (actor, key) => del(`actor/${actor.kar.type}/${actor.kar.id}/state/${key}`)

const actorGetAllState = (actor) => get(`actor/${actor.kar.type}/${actor.kar.id}/state`)

const actorSetStateMultiple = (actor, state = {}) => post(`actor/${actor.kar.type}/${actor.kar.id}/state`, state)

const actorRemoveAllState = (actor) => del(`actor/${actor.kar.type}/${actor.kar.id}/state`)

const broadcast = (path, args) => post(`system/broadcast/${path}`, args)

const shutdown = () => post('system/kill').then(() => agent.close())

function publish ({ topic, data, datacontenttype, dataschema, id, source, specversion = '1.0', subject, time, type }) {
  if (typeof topic === 'undefined') throw new Error('publish: must define "topic"')
  if (typeof id === 'undefined') throw new Error('publish: must define "id"')
  if (typeof source === 'undefined') throw new Error('publish: must define "source"')
  if (typeof type === 'undefined') throw new Error('publish: must define "type"')
  return post(`event/${topic}/publish`, { data, datacontenttype, dataschema, id, source, specversion, subject, time, type })
}

const subscribe = (topic, path, opts) => post(`event/${topic}/subscribe`, Object.assign({ path: `/${path}` }, opts))

const unsubscribe = (topic, opts) => post(`event/${topic}/unsubscribe`, opts)

const actorSubscribe = (actor, topic, path, params) => post(`event/${topic}/subscribe`, Object.assign({ path: `/${path}`, actorType: actor.kar.type, actorId: actor.kar.id }, params))

/***************************************************
 * End of public methods intended for application programming
 **************************************************/

const logger = truthy(process.env.KAR_VERBOSE) ? [morgan('--> :date[iso] :method :url', { immediate: true }), morgan('<-- :date[iso] :method :url :status - :response-time ms')] : []

const jsonParser = [
  parser.text({ type: '*/*' }), // parse to string first irrespective of content-type
  (req, _res, next) => {
    if (req._parsed) {
      next()
      return
    }
    req._parsed = true
    if (req.body.length > 0) {
      try {
        req.body = JSON.parse(req.body) // return parsed json
        next()
      } catch (err) {
        next(err) // forward errors to next middleware
      }
    } else {
      req.body = undefined // return undefined if request body is empty
      next()
    }
  }]

const errorHandler = [
  (err, req, res, next) => Promise.resolve()
    .then(() => {
      err.stack += `\n    at <kar> ${req.originalUrl}` // add request url to stack trace
      const body = {} // sanitize error object
      body.message = typeof err.message === 'string' ? err.message : typeof err === 'string' ? err : 'Internal Server Error'
      body.stack = typeof err.stack === 'string' ? err.stack : new Error(body.message).stack
      if (typeof err.errorCode === 'string') body.errorCode = err.errorCode
      return res.status(500).json(body) // return error
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

  router.use(jsonParser)

  // actor activation route
  router.get('/actor/:type/:id', (req, res, next) => Promise.resolve()
    .then(_ => {
      table[req.params.type] = table[req.params.type] || {}
      const actor = new (actors[req.params.type])(req.params.id)
      table[req.params.type][req.params.id] = actor
      table[req.params.type][req.params.id].kar = { type: req.params.type, id: req.params.id }
    }) // instantiate actor and add to index
    .then(_ => { // run optional activate callback
      if (typeof table[req.params.type][req.params.id].activate === 'function') return table[req.params.type][req.params.id].activate()
    })
    .then(_ => res.sendStatus(200)) // OK
    .catch(next))

  // actor deactivation route
  router.delete('/actor/:type/:id', (req, res, next) => Promise.resolve()
    .then(_ => { // run optional deactivate callback
      const actor = table[req.params.type][req.params.id]
      actor.kar.session = undefined
      if (typeof actor.deactivate === 'function') return actor.deactivate()
    })
    .then(_ => delete table[req.params.type][req.params.id]) // remove actor from index
    .then(_ => res.sendStatus(200)) // OK
    .catch(next))

  // method invocation route
  router.post('/actor/:type/:id/:session/:method', (req, res, next) => Promise.resolve()
    .then(_ => {
      const actor = table[req.params.type][req.params.id]
      if (req.params.method in actor) {
        actor.kar.session = req.params.session // NOTE: intentionally not cleared before return (could be nested call in same session)
        if (typeof actor[req.params.method] === 'function') {
          return req.body ? actor[req.params.method](...req.body) : actor[req.params.method]()
        }
        return actor[req.params.method]
      }
      throw new Error(`${req.params.method} is not defined on actor with type ${req.params.type} and id ${req.params.id}`)
    }) // invoke method on actor
    .then(result => res.json(result)) // stringify invocation result
    .catch(next)) // delegate error handling to postprocessor

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
  publish,
  subscribe,
  unsubscribe,
  actor: {
    proxy: actorProxy,
    tell: actorTell,
    call: actorCall,
    subscribe: actorSubscribe,
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
    broadcast,
    shutdown,
    h2c,
    logger,
    jsonParser,
    errorHandler
  }
}
