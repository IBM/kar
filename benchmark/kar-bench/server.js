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

const express = require('express')

const { sys } = require('kar-sdk')

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
app.post('/bench-text', (req, res) => {
  res.send(req.body)
})

// a route that accepts and returns application/json
app.post('/bench-json', (req, res) => {
  res.json(req.body)
})

app.post('/bench-text-one-way', (req, res) => {
  stamp = Date.now()
  res.send(stamp.toString())
})

class BenchActor {
  async simpleMethod() {
    return this.count
  }

  async timedMethod() {
    return Date.now().toString()
  }

  async activate() {
    this.count = 0
    console.log(`BenchActor activated`)
  }

  async deactivate() {
    console.log(`BenchActor deactivated`)
  }
}

app.post('/shutdown', async (_reg, res) => {
  console.log('Shutting down service')
  res.sendStatus(200)
  await sys.shutdown()
  server.close(() => process.exit())
})

// start server on port $KAR_APP_PORT
console.log('Starting server...')
app.use(sys.actorRuntime({ BenchActor }))
const server = app.listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')
