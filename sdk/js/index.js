const http = require('http') // for configuring http agent
const parser = require('body-parser') // for parsing http requests
const morgan = require('morgan') // for logging http requests and responses

// retry http requests up to 10 times on failure or 503 error
const fetch = require('fetch-retry')(require('node-fetch'), { retries: 10, retryOn: [503] })

// agent to keep connections to sidecar alive
const agent = new http.Agent({ keepAlive: true })

// url prefix for http requests to sidecar
const url = `http://localhost:${process.env.KAR_PORT || 3500}/kar/`

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

// check if string value is truthy
const truthy = s => s && s.toLowerCase() !== 'false' && s !== '0'

// public methods

// asynchronous service invocation, returns "OK" immediately
const send = (service, path, params) => post(`send/${service}/${path}`, params)

// synchronous service invocation, returns invocation result
const call = (service, path, params) => post(`call/${service}/${path}`, params)

// asynchronous actor invocation, returns "OK" immediately
const actorSend = (service, actor, path, params) => post(`session/${actor}/send/${service}/${path}`, params)

// synchronous actor invocation: returns invocation result
const actorCall = (service, actor, path, params) => post(`session/${actor}/call/${service}/${path}`, params)

// broadcast to all sidecars except for ours
const broadcast = (path, params) => post(`broadcast/${path}`, params)

// kill sidecar
const shutdown = () => get('kill')

// express middleware to log requests and responses if KAR_VERBOSE env variable is truthy
const logger = truthy(process.env.KAR_VERBOSE) ? [morgan('--> :date[iso] :method :url', { immediate: true }), morgan('<-- :date[iso] :method :url :status - :response-time ms')] : []

// express middleware to parse request bodies to json (non-strict, map empty body to undefined)
const jsonParser = [
  parser.text({ type: '*/*' }), // parse to string first irrespective of content-type
  (req, _res, next) => {
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

// exports
module.exports = {
  send,
  call,
  actor: { async: actorSend, call: actorCall },
  broadcast,
  shutdown,
  logger,
  jsonParser,
  errorHandler
}
