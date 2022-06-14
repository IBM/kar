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
const { v4: uuidv4 } = require('uuid')

const verbose = process.env.VERBOSE

class ActorTypeX {
	sleep (sleepTime) {
		return new Promise(resolve => setTimeout(resolve, sleepTime))
	}

	async activate(){
		Object.assign(this, await actor.state.getAll(this))
	}

	async randomNumber(){
		await this.sleep(20)
		return Math.random()
	}

	async doIt(timesRemaining){
		for(var i = 0; i < timesRemaining; i++){
			var myNum = await actor.call(this,
				actor.proxy("ActorTypeX", Math.random()),
				'randomNumber')
			console.log("My number: " + myNum)
		}
	}
}


// Server setup: register actors with KAR and start express
const app = express()
app.use(sys.actorRuntime({ ActorTypeX }))
app.listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')
