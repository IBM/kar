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
var t = require('../../transaction.js')
var c = require('../constants.js')

class DeliveryTxn extends t.Transaction {
  async activate () {
    await super.activate()
  }

  async getWarehouseDetails(wId) {
    const warehouse = actor.proxy('Warehouse', wId)
    return [warehouse, await actor.call(warehouse, 'getMultiple', ['ytd'])]
  }

  async getDistrictDetails(wId, dId) {
    const district = actor.proxy('District', wId + ':' + dId)
    return [district, await actor.call(district, 'getMultiple', ['nextOId', 'lastDlvrOrd'])]
  }

  async getCustomerDetails(wId, dId, cId) {
    const customer = actor.proxy('Customer', wId + ':' + dId + ':' + cId)
    const keys = ['balance', 'deliveryCnt']
    return [customer, await actor.call(customer, 'getMultiple', keys)]
  }

  async getOrderDetails(oId) {
    const order = actor.proxy('Order', oId)
    const keys = ['cId', 'orderLines', 'carrierId']
    return [order, await actor.call(order, 'getMultiple', keys)]
  }

  async getTotalOrderAmount(oDetails) {
    let totalAmt = 0
    for (let i in oDetails.orderLines.val) {
      totalAmt += oDetails.orderLines.val[i].amount
    }
    return totalAmt
  }

  async updateOrderDetails(oDetails, carrierId, deliveryDate) {
    let updatedODetails = Object.assign({}, oDetails)
    updatedODetails.carrierId.val = carrierId
    for (let i in oDetails.orderLines.val) {
      updatedODetails.orderLines.val[i].deliveryDate = deliveryDate
    }
    return updatedODetails
  }

  async updateCustomerDetails(cDetails, totalAmount) {
    let updatedCDetails = Object.assign({}, cDetails)
    updatedCDetails.balance.val += totalAmount
    updatedCDetails.deliveryCnt.val += 1
    return updatedCDetails
  }

  async startTxn(txn) {
    let actors = [], operations = [] /* Track all actors and their respective updates;
                                      perform the updates in an atomic txn. */
    const wDetails = await this.getWarehouseDetails(txn.wId)
    for (let i = 1; i <= c.NUM_DISTRICTS; i++) {
      let dId = 'd' + i
      const dDetails = await this.getDistrictDetails(txn.wId, dId)
      if (dDetails[1].nextOId.val == 1 || 
        dDetails[1].lastDlvrOrd.val == dDetails[1].nextOId.val - 1) {
        // This implies either no order was placed in this district
        // or all orders in the district are delivered; skip district
        continue
      }
      const orderId = txn.wId + ':' + dId + ':'+ 'o' + Number(dDetails[1].lastDlvrOrd.val+1)
      await actor.remove(actor.proxy('NewOrder', orderId))

      let dUpdate = {lastDlvrOrd: dDetails[1].lastDlvrOrd}
      dUpdate.lastDlvrOrd.val += 1
      actors.push(dDetails[0]), operations.push(dUpdate)

      const oDetails = await this.getOrderDetails(orderId)
      const updatedODetails = await this.updateOrderDetails(oDetails[1], txn.carrierId, txn.deliveryDate)
      actors.push(oDetails[0]), operations.push(updatedODetails)
      const totalOrderAmt = await this.getTotalOrderAmount(oDetails[1])

      const cDetails = await this.getCustomerDetails(txn.wId, dId, oDetails[1].cId)
      const updatedCDetails = await this.updateCustomerDetails(cDetails[1], totalOrderAmt)
      actors.push(cDetails[0]), operations.push(updatedCDetails)
    }
    await super.transact(actors, operations)
  }
}

exports.DeliveryTxn = DeliveryTxn