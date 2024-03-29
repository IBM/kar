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

const express = require('express')

if (!process.env.KAR_APP_PORT) {
  console.error('KAR_APP_PORT must be set. Aborting.')
  process.exit(1)
}

// create an express application
const app = express()

// parse request bodies with text/plain content type to json
app.use(express.text())

// parse request bodies with application/json content type to json
app.use(express.json())

// a route that accepts and returns text/plain
app.post('/helloText', (req, res) => {
  const msg = `Hello ${req.body}!`
  console.log(msg)
  res.send(msg)
})

// a route that accepts and returns application/json
app.post('/helloJson', (req, res) => {
  const msg = `Hello ${req.body.name}!`
  console.log(msg)
  res.json({ greetings: msg })
})

// a get route for health checks
app.get('/health', (req, res) => {
  console.log('I am healthy')
  res.send('I am healthy!')
})

// start server on port $KAR_APP_PORT
console.log('Starting greetings server...')
app.listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')
