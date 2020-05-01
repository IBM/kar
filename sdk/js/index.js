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
 * public methods
 ***************************************************/

/**
 * Asynchronous service invocation; returns "OK" immediately
 * @param {string} service The service to invoke.
 * @param {string} path The service endpoint to invoke.
 * @param {any} [params] The argument with which to invoke the service endpoint.
 */
const tell = (service, path, params) => post(`service/${service}/tell/${path}`, params)

/**
 * Synchronous service invocation; returns invocation result
 * @param {string} service The service to invoke.
 * @param {string} path The service endpoint to invoke.
 * @param {any} [params] The argument with which to invoke the service endpoint.
 * @returns The result returned by the target service.
 */
const call = (service, path, params) => post(`service/${service}/call/${path}`, params)

/**
 * Asynchronous actor invocation; returns "OK" immediately
 * @param {string} type The type of the target Actor.
 * @param {string} id The instance id of the target Actor.
 * @param {string} path The actor method to invoke.
 * @param {any} [params] The argument with which to invoke the actor method.
 */
const actorTell = (type, id, path, params) => post(`actor/${type}/${id}/tell/${path}`, params)

/**
 * Synchronous actor invocation; returns invocation result
 * @param {string} type The type of the target Actor.
 * @param {string} id The instance id of the target Actor.
 * @param {string} path The actor method to invoke.
 * @param {any} [params] The arguments with which to invoke the actor method.
 * @returns The result returned from the actor method
 */
const actorCall = (type, id, path, params) => post(`actor/${type}/${id}/call/${path}`, params)

/**
 * Synchronous actor invocation continuing the current session; returns invocation result
 * @param {string} type The type of the target Actor.
 * @param {string} id The instance id of the target Actor.
 * @param {string} session The session in which to invoke the method.
 * @param {string} path The actor method to invoke.
 * @param {any} [params] The arguments with which to invoke the actor method.
 * @returns The result returned from the actor method
 */

const actorCallInSession = (type, id, session, path, params) => post(`actor/${type}/${id}/call/${path}?session=${session}`, params)

/**
 * Cancel matching reminders for an Actor instance.
 * @param {string} type The type of the target Actor.
 * @param {string} id The instance id of the target Actor.
 * @param {string} [reminderId] The id of a specific reminder to cancel
 * @returns The number of reminders that were cancelled.
 */
const actorCancelReminder = (type, id, reminderId) => reminderId ? del(`actor/${type}/${id}/reminders/${reminderId}?nilOnAbsent=true`) : del(`actor/${type}/${id}/reminders`)

/**
 * Get matching reminders for an Actor instance.
 * @param {string} type The type of the target Actor.
 * @param {string} id  The instance id of the target Actor.
 * @param {string} [reminderId] The id of a specific reminder to cancel
 * @returns {array} An array of matching reminders
 */
const actorGetReminder = (type, id, reminderId) => reminderId ? get(`actor/${type}/${id}/reminders/${reminderId}?nilOnAbsent=true`) : get(`actor/${type}/${id}/reminders`)

/**
 * Schedule a reminder for an Actor instance.
 * @param {string} type The type of the target Actor.
 * @param {string} id The instance id of the target Actor.
 * @param {string} path The actor method to invoke when the reminder fires.
 * @param {{data?:any, deadline:Date, id:string, path:string, period?:string}} params A description of the desired reminder
 */
const actorScheduleReminder = (type, id, path, params) => post(`actor/${type}/${id}/reminders`, Object.assign({ path: `/${path}` }, params))

/**
 * Get one value from an Actor instance's state
 * @param {string} type The type of the Actor
 * @param {string} id The instance of of the Actor
 * @param {string} key The key to get from the instance's state
 * @returns The value associated with `key`
 */
const actorGetState = (type, id, key) => get(`actor/${type}/${id}/state/${key}?nilOnAbsent=true`)

/**
 * Set one value from an Actor instance's state
 * @param {string} type The type of the Actor
 * @param {string} id The instance of of the Actor
 * @param {string} key The key to set in the instance's state
 * @param {any} [value={}] The value to store
 */
const actorSetState = (type, id, key, value = {}) => put(`actor/${type}/${id}/state/${key}`, value)

/**
 * Delete a value from an Actor instance's state
 * @param {string} type The type of the Actor
 * @param {string} id The instance of of the Actor
 * @param {string} key The key to delete from the instance's state
 */
const actorDeleteState = (type, id, key) => del(`actor/${type}/${id}/state/${key}`)

/**
 * Get all key/value pairs in an Actor instance's state
 * @param {string} type The type of the Actor
 * @param {string} id The instance of of the Actor
 * @returns The state of the Actor
 */
const actorGetAllState = (type, id) => get(`actor/${type}/${id}/state`)

/**
 * Set multiple key/value pairs in an Actor instance's state
 * @param {string} type The type of the Actor
 * @param {string} id The instance of of the Actor
 * @param {Object.<string, any>} [state={}] The key/value pairs to store in the Actor's state
 * @returns The state of the Actor
 */
const actorSetStateMultiple = (type, id, state = {}) => post(`actor/${type}/${id}/state`, state)

/**
 * Delete all key/value pairs in an Actor instance's state
 * @param {string} type The type of the Actor
 * @param {string} id The instance of of the Actor
 */
const actorDeleteAllState = (type, id) => del(`actor/${type}/${id}/state`)

/**
 * Broadcast a message to all sidecars except for ours.
 * @param {string} path the path to invoke in each sidecar.
 * @param {any} params the parameters to pass to `path` when invoking it.
 */
const broadcast = (path, params) => post(`system/broadcast/${path}`, params)

/**
 * Kill this sidecar
 */
const shutdown = () => post('system/kill').then(() => agent.close())

/**
 * Publish a CloudEvent to a topic
 * @param {*} TODO: Document this API when it stabalizes
 */
function publish ({ topic, data, datacontenttype, dataschema, id, source, specversion = '1.0', subject, time, type }) {
  if (typeof topic === 'undefined') throw new Error('publish: must define "topic"')
  if (typeof id === 'undefined') throw new Error('publish: must define "id"')
  if (typeof source === 'undefined') throw new Error('publish: must define "source"')
  if (typeof type === 'undefined') throw new Error('publish: must define "type"')
  return post(`event/${topic}/publish`, { data, datacontenttype, dataschema, id, source, specversion, subject, time, type })
}

/**
 * Subscribe a Service endpoint to a topic.
 * @param {string} topic The topic to which to subscribe
 * @param {string} path The endpoint to invoke for each event received on the topic
 * @param {*} params TODO: Document expected structure
 */
const subscribe = (topic, path, params) => post(`event/${topic}/subscribe`, Object.assign({ path: `/${path}` }, params))

/**
 * Unsubscribe from a topic.
 * @param {string} topic The topic to which to subscribe
 * @param {*} params TODO: Document expected structure
 */
const unsubscribe = (topic, params) => post(`event/${topic}/unsubscribe`, params)

/**
 * Subscribe an Actor instance method to a topic.
 * @param {string} type The Actor type
 * @param {string} id The Actor instance id
 * @param {string} topic The topic to which to subscribe
 * @param {string} path The endpoint to invoke for each event received on the topic
 * @param {*} params TODO: Document expected structure
 */
const actorSubscribe = (type, id, topic, path, params) => post(`event/${topic}/subscribe`, Object.assign({ path: `/${path}`, actorType: type, actorId: id }, params))

/**
 * express middleware to log requests and responses if KAR_VERBOSE env variable is truthy
 */
const logger = truthy(process.env.KAR_VERBOSE) ? [morgan('--> :date[iso] :method :url', { immediate: true }), morgan('<-- :date[iso] :method :url :status - :response-time ms')] : []

/**
 * express middleware to parse request bodies to json (non-strict, map empty body to undefined)
 */
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

/**
 * express middleware to handle errors
 */
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

/**
 * Bind the Actor system to a specific Actor instance
 * @param {{kar:{session:string}}} actor The Actor instance being bound
 * @param {string} type The type of the Actor instance being bound
 * @param {string} id The id of the Actor instance being bound
 */
const kar = (actor, type, id) => ({
  id: id,
  tell: actorTell,
  tellSelf: (path, params) => actorTell(type, id, path, params),
  call: (type, id, path, params) => actorCallInSession(type, id, actor.kar.session, path, params),
  callSelf: (path, params) => actorCallInSession(type, id, actor.kar.session, path, params),
  get: key => actorGetState(type, id, key),
  set: (key, params) => actorSetState(type, id, key, params),
  setMultiple: (params) => actorSetStateMultiple(type, id, params),
  delete: key => actorDeleteState(type, id, key),
  getAll: () => actorGetAllState(type, id),
  deleteAll: () => actorDeleteAllState(type, id),
  cancelReminder: (reminderId) => actorCancelReminder(type, id, reminderId),
  getReminder: (reminderId) => actorGetReminder(type, id, reminderId),
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
      const actor = new (actors[req.params.type])(req.params.id)
      table[req.params.type][req.params.id] = actor
      table[req.params.type][req.params.id].kar = kar(actor, req.params.type, req.params.id)
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
      actor.kar.session = session
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
      setMultiple: actorSetStateMultiple,
      delete: actorDeleteState,
      getAll: actorGetAllState,
      deleteAll: actorDeleteAllState
    }
  },
  actorProxy: function (type, id) { const proxy = {}; proxy.kar = kar(proxy, type, id); return proxy },
  actorRuntime,
  broadcast,
  shutdown,
  logger,
  jsonParser,
  errorHandler
}
