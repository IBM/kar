/*
 * Copyright IBM Corporation 2020,2021
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

// retry http requests up to 10 times over 10s
const fetch = require('fetch-retry')(require('node-fetch'), { retries: 10 })

if (!process.env.KAR_RUNTIME_PORT) {
  console.error('KAR_RUNTIME_PORT must be set. Aborting.')
  process.exit(1)
}

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
