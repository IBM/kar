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
const { actor, sys } = require('kar-sdk')
var gp = require('../../generic_participant.js')
var c = require('../constants.js')
const verbose = process.env.VERBOSE

class ItemStock extends gp.GenericParticipant {
  async activate() {
    const that = await super.activate()
    this.itemId = that.itemId || this.kar.id
    this.wId = that.wId || this.kar.id.split(':')[1]
    this.name = that.name || c.DEFAULT_ITEM_NAME
    this.price = that.price || c.DEFAULT_ITEM_PRICE
    this.quantity = that.quantity || await super.createVal(c.DEFAULT_QUANTITY)
    this.ytd = that.ytd || await super.createVal(0)
    this.orderCnt = that.orderCnt || await super.createVal(0)
    this.remoteCnt = that.remoteCnt || await super.createVal(0)
    this.data = that.data
  }

  async addNewItem(item) {
    this.wId = item.wId
    this.name = item.name || c.DEFAULT_ITEM_NAME
    this.price = item.price || c.DEFAULT_ITEM_PRICE
    await actor.state.setMultiple(this, {wId : this.wId, name: this.name, price: this.price})
  }

  async prepareNewOrder(txnId) {
    const keys = ['price', 'name', 'quantity', 'ytd', 'orderCnt']
    return await this.prepare(txnId, keys)
  }

  async commitNewOrder(txnId, decision, update) {
    return await this.commit(txnId, decision, update)
  }
}

// Server setup: register actors with KAR and start express
const app = express()
app.use(sys.actorRuntime({ ItemStock }))
app.listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')