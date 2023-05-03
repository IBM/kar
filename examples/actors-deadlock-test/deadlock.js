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
const { actor, sys } = require('kar-sdk')
const { v4: uuidv4 } = require('uuid')

const verbose = process.env.VERBOSE

class ActorTypeA {
  async activate () {
    Object.assign(this, await actor.state.getAll(this))
  }

  sleep () {
    return new Promise(resolve => setTimeout(resolve, 500))
  }

  async doIt(){
	console.log("ActorTypeA has started.")
	console.log("Sleeping for 500ms.")
	await this.sleep()
	console.log("Calling B.finish().")
	await actor.call(this, actor.proxy('ActorTypeB', "B"), 'finish', Math.random())
	console.log("Finished.")
  }

  async finish(testArg) {
	console.log("ActorTypeA: finish has been called with arg " + testArg + ".")
  }
}

class ActorTypeB {
  async activate () {
    Object.assign(this, await actor.state.getAll(this))
  }

  sleep () {
    return new Promise(resolve => setTimeout(resolve, 200))
  }

  async doIt(){
	console.log("ActorTypeB has started.")
	console.log("Sleeping for 200ms.")
	await this.sleep()
	console.log("Calling A.finish().")
	await actor.call(this, actor.proxy('ActorTypeA', "A"), 'finish', Math.random())
	console.log("Finished.")
  }

  async finish() {
	console.log("ActorTypeB: finish has been called with arg " + testArg + ".")

  }
}

class Tester {
  async activate () {
    Object.assign(this, await actor.state.getAll(this))
  }

  async startTest(){
	actor.tell(actor.proxy("ActorTypeA", "A"), "doIt")
	actor.tell(actor.proxy("ActorTypeB", "B"), "doIt")
  }
}

// Server setup: register actors with KAR and start express
const app = express()
app.use(sys.actorRuntime({ ActorTypeA, ActorTypeB, Tester }))
app.listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')
