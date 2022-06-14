/*
 * Copyright IBM Corporation 2020,2022
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

const verbose = process.env.VERBOSE

const names = ["A", "B", "C"]

class Starter {
	async activate(){
		Object.assign(this, await actor.state.getAll(this))
	}

	async start() {
		await actor.tell(actor.proxy('TestActor', 'A'), 'doIt',
			500, 'A', 'B')
		await actor.tell(actor.proxy('TestActor', 'B'), 'doIt', 1000, 'B', 'A')
	}
}

class TestActor {
  async activate(){
	Object.assign(this, await actor.state.getAll(this))
  }
  sleep (sleepTime) {
	return new Promise(resolve => setTimeout(resolve, sleepTime))
  }
  async doIt (sleepTime, name, nextName) {
	console.log("Actor " + name + " has been called.")
	await this.sleep(sleepTime)
	console.log("Actor " + name + " calling " + nextName + ".finish()")
  	await actor.call(this, actor.proxy('TestActor', nextName), 'finish', nextName)
  }
  async finish(name){
	console.log("Actor " + name + " is finished!")
  }
}

// Server setup: register actors with KAR and start express
const app = express()
app.use(sys.actorRuntime({ TestActor, Starter }))
app.listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')
