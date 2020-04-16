const express = require('express')
const http2 = require('http2')
const parser = require('body-parser') // for parsing http requests
const morgan = require('morgan') // for logging http requests and responses
const spdy = require('spdy')

const agent = http2.connect(`http://localhost:${process.env.KAR_PORT || 3500}`)

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

// http get: parse response body
function get (api) {
  return fetch(url + api, { headers, agent }).then(parse)
}

// http get: parse response body
function del (api) {
  return fetch(url + api, { method: 'DELETE', headers, agent }).then(parse)
}

// check if string value is truthy
const truthy = s => s && s.toLowerCase() !== 'false' && s !== '0'

// public methods

// asynchronous service invocation, returns "OK" immediately
const tell = (service, path, params) => post(`service/${service}/tell/${path}`, params)

// synchronous service invocation, returns invocation result
const call = (service, path, params) => post(`service/${service}/call/${path}`, params)

// asynchronous actor invocation, returns "OK" immediately
const actorTell = (type, id, path, params) => post(`actor/${type}/${id}/tell/${path}`, params)

// synchronous actor invocation: returns invocation result
const actorCall = (type, id, path, params) => post(`actor/${type}/${id}/call/${path}`, params)
const actorCallInSession = (type, id, session, path, params) => post(`actor/${type}/${id}/call/${path}?session=${session}`, params)

// reminder operations
const actorCancelReminder = (type, id, params = {}) => del(`actor/${type}/${id}/reminder`, params)
const actorGetReminder = (type, id, params = {}) => get(`actor/${type}/${id}/reminder`, params)
const actorScheduleReminder = (type, id, path, params) => post(`actor/${type}/${id}/reminder`, Object.assign({ path: `/${path}` }, params))

// actor state operations
const actorGetState = (type, id, key) => get(`actor/${type}/${id}/state/${key}`)
const actorSetState = (type, id, key, params = {}) => post(`actor/${type}/${id}/state/${key}`, params)
const actorDeleteState = (type, id, key) => del(`actor/${type}/${id}/state/${key}`)
const actorGetAllState = (type, id) => get(`actor/${type}/${id}/state`)
const actorDeleteAllState = (type, id) => del(`actor/${type}/${id}/state`)

// broadcast to all sidecars except for ours
const broadcast = (path, params) => post(`system/broadcast/${path}`, params)

// kill sidecar
const shutdown = () => get('system/kill').then(() => agent.close())

// pubsub
const publish = (topic, params) => post(`event/${topic}/publish`, params)
const subscribe = (topic, path, params) => post(`event/${topic}/subscribe`, Object.assign({ path: `/${path}` }, params))
const unsubscribe = (topic, params) => post(`event/${topic}/unsubscribe`, params)

const actorSubscribe = (type, id, topic, path, params) => post(`event/${topic}/subscribe`, Object.assign({ path: `/${path}`, actorType: type, actorId: id }, params))

// express middleware to log requests and responses if KAR_VERBOSE env variable is truthy
const logger = truthy(process.env.KAR_VERBOSE) ? [morgan('--> :date[iso] :method :url', { immediate: true }), morgan('<-- :date[iso] :method :url :status - :response-time ms')] : []

// express middleware to parse request bodies to json (non-strict, map empty body to undefined)
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

// express middleware to handle errors
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

const sys = (type, id) => ({
  id: id,
  tell: (method, params) => actorTell(type, id, method, params),
  get: key => actorGetState(type, id, key),
  set: (key, params) => actorSetState(type, id, key, params),
  delete: key => actorDeleteState(type, id, key),
  getAll: () => actorGetAllState(type, id),
  deleteAll: () => actorDeleteAllState(type, id),
  cancelReminder: params => actorCancelReminder(type, id, params),
  getReminder: params => actorGetReminder(type, id, params),
  scheduleReminder: (path, params) => actorScheduleReminder(type, id, path, params)
})

const table = {} // live actor instances: table[actorType][actorId]

function actorRuntime (actors) {
  const router = express.Router()

  router.use(jsonParser)

  // actor activation route
  router.get('/actor/:type/:id', (req, res, next) => Promise.resolve()
    .then(_ => {
      table[req.params.type] = table[req.params.type] || {}
      table[req.params.type][req.params.id] = new (actors[req.params.type])(req.params.id)
      table[req.params.type][req.params.id].sys = sys(req.params.type, req.params.id)
    }) // instantiate actor and add to index
    .then(_ => { // run optional activate callback
      if (typeof table[req.params.type][req.params.id].activate === 'function') return table[req.params.type][req.params.id].activate()
    })
    .then(_ => res.sendStatus(200)) // OK
    .catch(next))

  // actor deactivation route
  router.delete('/actor/:type/:id', (req, res, next) => Promise.resolve()
    .then(_ => { // run optional deactivate callback
      if (typeof table[req.params.type][req.params.id].deactivate === 'function') return table[req.params.type][req.params.id].deactivate()
    })
    .then(_ => delete table[req.params.type][req.params.id]) // remove actor from index
    .then(_ => res.sendStatus(200)) // OK
    .catch(next))

  // method invocation route
  router.post('/actor/:type/:id/:session/:method', (req, res, next) => Promise.resolve()
    .then(_ => {
      const session = req.params.session
      const actor = table[req.params.type][req.params.id]
      actor.actors = new Proxy({}, {
        get: function (_, type) {
          return new Proxy({}, {
            get: function (_, id) {
              return new Proxy({}, {
                get: function (_, method) {
                  return params => actorCallInSession(type, id, session, method, params)
                }
              })
            }
          })
        }
      })
      if (req.params.method in actor) {
        if (typeof actor[req.params.method] === 'function') {
          return actor[req.params.method](req.body)
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

const actors = new Proxy({}, {
  get: function (_, type) {
    return new Proxy({}, {
      get: function (_, id) {
        return new Proxy({}, {
          get: function (_, method) {
            if (method === 'sys') {
              return sys(type, id)
            } else {
              return params => actorCall(type, id, method, params)
            }
          }
        })
      }
    })
  }
})

// h2c protocol wrapper
const h2c = app => spdy.createServer({ spdy: { plain: true, ssl: false, connection: { maxStreams: 262144 } } }, app).setTimeout(0)

// exports
module.exports = {
  h2c,
  tell,
  call,
  publish,
  subscribe,
  unsubscribe,
  actor: {
    tell: actorTell,
    call: actorCall,
    subscribe: actorSubscribe,
    cancelReminder: actorCancelReminder,
    getReminder: actorGetReminder,
    scheduleReminder: actorScheduleReminder,
    state: {
      get: actorGetState,
      set: actorSetState,
      delete: actorDeleteState,
      getAll: actorGetAllState,
      deleteAll: actorDeleteAllState
    }
  },
  actors,
  actorRuntime,
  broadcast,
  shutdown,
  logger,
  jsonParser,
  errorHandler
}
