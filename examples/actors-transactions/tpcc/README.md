<!--
# Copyright IBM Corporation 2020,2021
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
-->

# TPC-C implementation in KAR
This example runs [TPC-C](http://tpc.org/tpc_documents_current_versions/pdf/tpc-c_v5.11.0.pdf) transactional benchmarking in KAR. TPC-C benchmarking application mimics an ecommerce application where customers place new orders, make payments, etc,. TPC-C TPC-C consists of 5 types of transactions: New Order, Payment, Delivery, Order Status, and Stock Level; the first three are read-write transactions and the last two are read-only. New Order and Payment transactions form 88% (44% each) of the workload and the rest all form 4% of the workload. The first 4 transactions need strong consistency guarantees and the last one can be of relaxed consistency.

## Actors
There are 5 different types of transactional actors (for 5 types of transactions) and 6 different TPC-C application specific actors. The application specific actors are Warehouse, District, Customer, ItemStock, Order, and New Order, each representing a table in TPC-C. The ItemStock represents the combined tables Item and Stock in TPC-C; this implementation does not implement an actor equibvalent to the History table in TPC-C. The default number of warehouses are 10, as defined in constants.js.

### Deploying actors
While all actors can be created either in a single process or different processes, tn the implementation
all application specific actors are imported into tpcc_actors.js and all transaction actors are imported into tpcc_transactions.js. From tpcc path, run the below commands in two different terminals to start the application and transaction actors:
```

kar run -app tpcc -app_port 8081 -actors Warehouse,District,Customer,NewOrder,Order,ItemStock node actors/tpcc_actors.js

kar run -app tpcc  -app_port 8082 -actors PaymentTxn,NewOrderTxn,OrderStatusTxn,DeliveryTxn,StockLevelTxn node transactions/tpcc_transactions.js

```
## TPC-C Client
The tpcc_client.js file simulates multiple clients sending transaction requests. The three field NUM_TXNS, WARM_UP_TXNS, and CONCURRENCY can be changed to send more overall transactions or perform more warmup transactions or to increase the concurrent requests sent. The client, after executing NUM_TXNS number of transactions, computes the overall throughput and average end-to-end latency of transactions and logs the result on the console. The client also logs how many of each type of transaction succeeded. To run the client, from tpcc path, execute the below command:

```
kar run -app tpcc node tpcc_client.js
```

The tests.js file compiles consistency tests for each type of transaction and can be used to test the expected behaviour if any code is changed. The test file can be executed using the following command:
```
kar run -app tpcc node tests.js
```

## Transactional Framework
To provide transactional semantic (ACID properties), 2 Phase Commit is implemented. Each transaction actor, upon receiving a client request, prepares all the involved actors, and if all of those actors accept the transaction, the transaction is committed and otherwise aborted. To reduce cross-actor communication, prepare phase is also the read phase of a transaction. Each participating actor called a *participant* 'locks' the fields accessed by a transaction. A field within a participant can have a single read-write transaction accessing it or any number of read-only transactions accessing it. The value of the fields are not updated until the commit phase.

The txn_framework outside of tpcc folder implements the concurrency control logic and sending the prepare/commit messages to the participants. The txn_framework consists of two generic files: transactions.js and txn_participant.js. The transaction.js implements sending prepare and commit messages to all the participants. A TPC-C transaction can invoke these methods after identifying the involed participants (for prepare) or the necessary transaction specific updates (for commit). The txn_participant.js implements a participant's concurrency control logic, and provides a generic prepare and commit methods that can be called by a transaction actor. TPC-C actors inherit this class and may over-write methods if needed.