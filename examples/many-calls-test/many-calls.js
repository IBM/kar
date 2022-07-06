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

const numNames = 10

class ActorTypeX {
	sleep (sleepTime) {
		return new Promise(resolve => setTimeout(resolve, sleepTime))
	}

	async activate(){
		Object.assign(this, await actor.state.getAll(this))
	}

	async methodOnlyCalledByTells(){
		await this.sleep(5000)
	}

	async doItA(timesRemaining, originalName){
		await this.sleep(20)
		if(timesRemaining <= 0){
			//console.log("Done!")
			return
		}
		await actor.call(this,
			actor.proxy('ActorTypeX', originalName + (timesRemaining%numNames)), 
			'doItC',
			timesRemaining-1, originalName)
	}
	async doItB(timesRemaining, originalName){
		await this.sleep(20)
		if(timesRemaining <= 0){
			//console.log("Done!")
			return
		}
		await actor.call(this,
			actor.proxy('ActorTypeX', originalName + (timesRemaining%numNames)), 
			'doItC',
			timesRemaining-1, originalName)
	}
	async doItC(timesRemaining, originalName){
		await this.sleep(20)
		if(timesRemaining <= 0){
			//console.log("Done!")
			return
		}
		var coinFlip = Math.random();
		var letter = "A"
		if(coinFlip > 0.5) letter = "B"
		await actor.tell(actor.proxy('ActorTypeX', ""+Math.random()), 'methodOnlyCalledByTells')
		await actor.tell(actor.proxy('ActorTypeX', ""+Math.random()), 'doItC')
		await actor.call(this,
			actor.proxy('ActorTypeX', originalName + (timesRemaining%numNames)), 
			'doIt' + letter,
			timesRemaining-1, originalName)
	}
}

/*class ActorTypeY {
  async activate(){
	Object.assign(this, await actor.state.getAll(this))
  }
  sleep (sleepTime) {
	return new Promise(resolve => setTimeout(resolve, sleepTime))
  }
  async tellDoIt (sleepTime, name, nextName) {
	console.log("Actor " + name + " has been called.")
	await this.sleep(sleepTime)
	console.log("Actor " + name + " calling " + nextName + ".finish()")
  	await actor.call(this, actor.proxy('TestActor', nextName), 'finish', nextName)
  }
  async finish(name){
	console.log("Actor " + name + " is finished!")
  }
}*/

// Server setup: register actors with KAR and start express
const app = express()
app.use(sys.actorRuntime({ ActorTypeX }))
app.listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')
