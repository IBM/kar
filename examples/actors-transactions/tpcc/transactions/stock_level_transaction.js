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

const { actor, sys } = require('kar-sdk')
var t = require('../../txn_framework/transaction.js')

class StockLevelTxn extends t.Transaction {
  async activate () {
    await super.activate()
  }

  async getDistrictDetails(wId, dId) {
    const district = actor.proxy('District', wId + ':' + dId)
    return [district, await actor.call(district, 'getMultiple', ['nextOId'])]
  }

  async startTxn(txn) {
    const dDetails = await this.getDistrictDetails(txn.wId, txn.dId)
    let lowStockCnt = 0, orderPromises = [], stockPromises = []
    const index = dDetails[1].nextOId.val - 1
    for (let i = index; i > index - 20 && i > 0; i-- ) {
      const order = actor.proxy('Order', txn.wId + ':' + txn.dId + ':' + 'o' + Number(i))
      orderPromises.push(actor.call(order, 'get', 'orderLines'))
    }
    const oDetails = await Promise.all(orderPromises)
    for (let i in oDetails) {
      const ol = oDetails[i].val
      for (let key in ol) {
        const stock = actor.proxy('ItemStock', ol[key].itemId + ':' + ol[key].supplyWId)
        stockPromises.push(actor.call(stock, 'get', 'quantity'))
      }
    }
    const stockDetails = await Promise.all(stockPromises)
    for (let i in stockDetails) {
      if (stockDetails[i].val < txn.threshold) { lowStockCnt ++ }
    }
    return {decision:true, lowStockCnt:lowStockCnt}
  }
}

exports.StockLevelTxn = StockLevelTxn