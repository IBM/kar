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

class OrderStatusTxn extends t.Transaction {
  async activate () {
    const that = await super.activate()
  }

  async getCustomerDetails(wId, dId, cId) {
    const customer = actor.proxy('Customer', wId + ':' + dId + ':' + cId)
    const keys = ['balance', 'name', 'lastOId']
    return [customer, await actor.call(customer, 'getMultiple', keys)]
  }

  async getOrderDetails(oId) {
    const order = actor.proxy('Order', oId)
    const keys = ['entryDate', 'carrierId', 'orderLines']
    return [order, await actor.call(order, 'getMultiple', keys)]
  }

  async startTxn(txn) {
    const cDetails = await this.getCustomerDetails(txn.wId, txn.dId, txn.cId)
    const oDetails = await this.getOrderDetails(cDetails[1].lastOId.lastOId)
    return oDetails[1]
  }
}

exports.OrderStatusTxn = OrderStatusTxn