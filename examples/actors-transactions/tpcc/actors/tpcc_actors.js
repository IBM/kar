const express = require('express')
const { actor, sys } = require('kar-sdk')
var w = require('./warehouse.js')
Warehouse = w.Warehouse
var d = require('./district.js')
District = d.District
var c = require('./customer.js')
Customer = c.Customer
var no = require('./order.js')
NewOrder = no.NewOrder
var o = require('./order.js')
Order = o.Order
var is = require('./stock.js')
ItemStock = is.ItemStock

// Server setup: register actors with KAR and start express
const app = express()
app.use(sys.actorRuntime({ Warehouse, District, Customer, NewOrder, Order, ItemStock  }))
sys.h2c(app).listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')