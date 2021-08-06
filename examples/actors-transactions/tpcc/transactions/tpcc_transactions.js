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
var not = require('./new_order_transaction.js')
NewOrderTxn = not.NewOrderTxn 
var pt = require('./payment_transaction.js')
PaymentTxn = pt.PaymentTxn
var ost = require('./order_status_transaction.js')
OrderStatusTxn = ost.OrderStatusTxn
var dt = require('./delivery_transaction.js')
DeliveryTxn = dt.DeliveryTxn

// Server setup: register actors with KAR and start express
const app = express()
app.use(sys.actorRuntime({ NewOrderTxn, PaymentTxn, OrderStatusTxn, DeliveryTxn }))
app.listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')