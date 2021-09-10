const express = require('express')
const { sys } = require('kar-sdk')
const w = require('./warehouse.js')
const Warehouse = w.Warehouse
const d = require('./district.js')
const District = d.District
const c = require('./customer.js')
const Customer = c.Customer
const no = require('./order.js')
const NewOrder = no.NewOrder
const o = require('./order.js')
const Order = o.Order
const is = require('./stock.js')
const ItemStock = is.ItemStock

// Server setup: register actors with KAR and start express
const app = express()
app.use(sys.actorRuntime({ Warehouse, District, Customer, NewOrder, Order, ItemStock }))
sys.h2c(app).listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')
