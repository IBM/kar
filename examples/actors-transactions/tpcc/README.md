This example runs [TPC-C](http://tpc.org/tpc_documents_current_versions/pdf/tpc-c_v5.11.0.pdf) transactional benchmarking in KAR. TPC-C benchmarking application mimics an ecommerce application where customers place new orders, make payments, etc,. TPC-C TPC-C consists of 5 types of transactions: New Order, Payment, Delivery, Order Status, and Stock Level; the first three are read-write transactions and the last two are read-only. New Order and Payment transactions form 88% (44% each) of the workload and the rest all form 4% of the workload. The first 4 transactions need strong consistency guarantees and the last one can be of relaxed consistency.

## Actors
There are 5 different types of transactional actors (for 5 types of transactions) and 6 different TPC-C application specific actors. The application specific actors are Warehouse, District, Customer, ItemStock, Order, and New Order, each representing a table in TPC-C. The ItemStock represents the combined tables Item and Stock in TPC-C; this implementation does not implement an actor equibvalent to the History table in TPC-C. The default number of warehouses are 10, as defined in constants.js.

# Deploying actors
While all actors can be created either in a single process or different processes, tn the implementation
all application specific actors are imported into tpcc_actors.js and all transaction actors are imported into tpcc_transactions.js. From tpcc path, run the below commands in two different terminals to start the application and transaction actors:
```

kar run -app tpcc -app_port 8081 -actors Warehouse,District,Customer,NewOrder,Order,ItemStock node actors/tpcc_actors.js

kar run -app tpcc  -app_port 8082 -actors PaymentTxn,NewOrderTxn,OrderStatusTxn,DeliveryTxn,StockLevelTxn node transactions/tpcc_transactions.js

```
## TPC-C Client
The tpcc_client.js file simulates multiple clients sending transaction requests. The three field NUM_TXNS, WARM_UP_TXNS, and CONCURRENCY can be changed to send more overall transactions or perform more warmup transactions or to increase the concurrent requests sent. The client, after executing NUM_TXNS number of transactions, computes the throughput and average latency of a transaction and logs the result on the console. The client also logs how many of each type of transaction succeeded. To run the client, from tpcc path, execute the below command:

```
kar run -app tpcc node tpcc_client.js
```