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
const { sys } = require('kar-sdk')
const not = require('./new_order_transaction.js')
const NewOrderTxn = not.NewOrderTxn
const pt = require('./payment_transaction.js')
const PaymentTxn = pt.PaymentTxn
const ost = require('./order_status_transaction.js')
const OrderStatusTxn = ost.OrderStatusTxn
const dt = require('./delivery_transaction.js')
const DeliveryTxn = dt.DeliveryTxn
const slt = require('./stock_level_transaction.js')
const StockLevelTxn = slt.StockLevelTxn

// Server setup: register actors with KAR and start express
const app = express()
app.use(sys.actorRuntime({ NewOrderTxn, PaymentTxn, OrderStatusTxn, DeliveryTxn, StockLevelTxn }))
sys.h2c(app).listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')
