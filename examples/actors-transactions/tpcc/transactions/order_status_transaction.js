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

class PaymentTxn extends t.Transaction {
  async activate () {
    const that = await super.activate()
  }

  async getWarehouseDetails(wId) {
    const warehouse = actor.proxy('Warehouse', wId)
    return [warehouse, await actor.call(warehouse, 'getMultiple', ['ytd'])]
  }

  async getDistrictDetails(wId, dId) {
    const district = actor.proxy('District', wId + ':' + dId)
    return [district, await actor.call(district, 'getMultiple', ['ytd'])]
  }

  async getCustomerDetails(wId, dId, cId) {
    const customer = actor.proxy('Customer', wId + ':' + dId + ':' + cId)
    await actor.call(customer, 'addCustomerToDistrict', dId, wId)
    const keys = ['balance', 'ytdPayment', 'paymentCnt']
    return [customer, await actor.call(customer, 'getMultiple', keys)]
  }

  async updateCustomerDetails(cDetails, amount) {
    let updatedCDetails = Object.assign({}, cDetails)
    // Update customer details based on txn payment.
    updatedCDetails.balance.balance = cDetails.balance.balance - amount
    updatedCDetails.ytdPayment.ytdPayment = cDetails.ytdPayment.ytdPayment + amount
    updatedCDetails.paymentCnt.paymentCnt = cDetails.paymentCnt.paymentCnt + 1
    return updatedCDetails
  }

  async startTxn(txn) {
    let actors = [], operations = [] /* Track all actors and their respective updates;
                                      perform the updates in an atomic txn. */
    const wDetails = await this.getWarehouseDetails(txn.wId)
    const dDetails = await this.getDistrictDetails(txn.wId, txn.dId)
    const cDetails = await this.getCustomerDetails(txn.wId, txn.dId, txn.cId)

    actors.push(wDetails[0]), operations.push({ytd : {ytd: wDetails[1].ytd.ytd + txn.amount,
                                              v : wDetails[1].ytd.v }})
    actors.push(dDetails[0]), operations.push({ytd : {ytd: dDetails[1].ytd.ytd + txn.amount,
                                                v : dDetails[1].ytd.v }})
    const updatedCDetails = await this.updateCustomerDetails(cDetails[1], txn.amount)
    actors.push(cDetails[0]), operations.push(updatedCDetails)

    await super.transact(actors, operations)
  }
}

// Server setup: register actors with KAR and start express
// const app = express()
// app.use(sys.actorRuntime({ PaymentTxn }))
// app.listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')

exports.PaymentTxn = PaymentTxn