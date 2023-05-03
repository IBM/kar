/*
 * Copyright IBM Corporation 2020,2023
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

 const fetch = require('node-fetch')

function url (service, route) {
  return `http://127.0.0.1:${process.env.KAR_RUNTIME_PORT}/kar/v1/service/${service}/call/${route}`
}

async function post (body) {
  console.log('sending request with body', body)
  const res = await fetch(url('echo', 'echo'), {
    method: 'POST',
    body,
    headers: { 'Content-Type': 'text/plain' }
  })
  const text = await res.text()
  console.log('received response with body', text)
}

async function main () {
  await post('Joe')
  await post('Jack')
  await post('Josh')
}

main()
