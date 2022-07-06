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

class TestActor {
  async activate(){
	Object.assign(this, await actor.state.getAll(this))
  }
  /*async hi(triesRemaining, name){
  	//let name = "hullo"
	console.log(name+triesRemaining)
	if(triesRemaining == 0){
		console.log("Success!")
		return
	}
	let newName = ""+Math.random()
	console.log(newName)
  	await actor.call(actor.proxy('TestActor', newName), 'hi', triesRemaining-1, newName)
  }*/
  async doIt (triesRemaining, name) {
  	console.log(name + triesRemaining)
	let storedNum = await actor.state.get(this, 'num') || -1
	console.log("Actor " + name + " stored num: " + storedNum)
  	if(triesRemaining == 0){
		console.log("Success!")
		return
	}
	let newName = names[Math.floor(Math.random()*names.length)]
	console.log("\tAbout to call" + newName)
	await actor.state.set(this, 'num', triesRemaining)
  	await actor.call(this, actor.proxy('TestActor', newName), 'doIt', triesRemaining-1, newName)
  }

  noDeadlock(num){
	console.log("Num: " + num)
	actor.call(this, actor.proxy('TestActor', names[num+1]), 'noDeadlock', num+1)
  }
}

// Server setup: register actors with KAR and start express
const app = express()
app.use(sys.actorRuntime({ TestActor }))
app.listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')
