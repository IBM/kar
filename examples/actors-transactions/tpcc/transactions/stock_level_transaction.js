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
var t = require('../../transaction.js')
var c = require('../constants.js')
const verbose = process.env.VERBOSE

class StockLevelTxn extends t.Transaction {
  async activate () {
    const that = await super.activate()
  }

  async getDistrictDetails(wId, dId) {
    const district = actor.proxy('District', wId + ':' + dId)
    return [district, await actor.call(district, 'getMultiple', ['nextOId'])]
  }

  async getOrderDetails(wId, dId, oId) {
    const order = actor.proxy('Order', wId + ':' + dId + ':' + oId)
    return [order, await actor.call(order, 'getMultiple', ['orderLines'])]
  }

  async getItemDetails(itemId, supplyWId) {
    const itemStock = actor.proxy('ItemStock', itemId + ':' + supplyWId)
    return [itemStock, await actor.call(itemStock, 'getMultiple', ['quantity'])]
  }

  async startTxn(txn) {
    const dDetails = await this.getDistrictDetails(txn.wId, txn.dId)
    let lowStockCnt = 0
    const index = dDetails[1].nextOId.nextOId - 1
    for (let i = index; i > index - 20 && i > 0; i-- ) {
      const oDetails = await this.getOrderDetails(txn.wId, txn.dId, 'o' + Number(i))
      const olines = oDetails[1].orderLines.orderLines
      for (let key in olines) {
        const itemId = olines[key].itemId
        const supplyWId = olines[key].supplyWId
        const itemDetails = await this.getItemDetails(itemId, supplyWId)
        if (itemDetails[1].quantity.quantity < txn.threshold) { lowStockCnt ++ }
      }
    }
    return lowStockCnt
  }
}

exports.StockLevelTxn = StockLevelTxn