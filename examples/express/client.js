// retry http requests up to 10 times over 10s
const fetch = require('fetch-retry')(require('node-fetch'), { retries: 10 })

// request url for a given KAR service and route on that service
function url (service, route) {
  return `http://127.0.0.1:${process.env.KAR_RUNTIME_PORT}/kar/v1/service/${service}/call/${route}`
}

// main method
async function main () {
  // an http request with text/plain content type
  const res1 = await fetch(url('greeter', 'helloText'), {
    method: 'POST',
    body: 'John Doe',
    headers: { 'Content-Type': 'text/plain' }
  })

  // parse response body to text
  const text = await res1.text()
  console.log(text)

  // an http request with application/json content type
  const res2 = await fetch(url('greeter', 'helloJson'), {
    method: 'POST',
    body: JSON.stringify({ name: 'Jane Doe' }),
    headers: { 'Content-Type': 'application/json' }
  })

  // parse response body to json
  const obj = await res2.json()
  console.log(obj.greetings)
}

// invoke main
main()
