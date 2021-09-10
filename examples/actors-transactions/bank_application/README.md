# A simple money transfer transaction in KAR

This is a simple transactional application that transfers money between 2 (or more) accounts. The account_server.js file defines an Account actor type and a MoneyTransfer actor type, which is a transactional actor. Each transaction has two phases: prepare and commit. If all accounts agree to prepare, then the transaction is committed and otherwise aborted. The Account actor tracks cumulative credits and debits of ongoing, uncommitted transactions, and updates the account balance only in the commit phase. 

## Client
The account_client.js file generates transactional workloads by creating money transfer transaction between 2 (or more) accounts out of 1000 accounts. The client creates 200 transactions with 20 concurrent threads. All of these parameters are defined in the account_client.js file and can be modified:

```
const NUM_ACCTS = 1000
const ACCTS_PER_TXN = 2
const NUM_TXNS = 200
const CONCURRENCY = 20
```

After executing all the transactions, the client prints on console the average latency of the transactions and the throughput, along with the number of successfully committed transactions.

## Deploying money transfer application
After ensuring the KAR deployment works (instructions can be found [here](https://github.com/IBM/kar/blob/main/docs/getting-started.md)), run the below commands in two different windows starting from the bank_application directory:

```

kar run -app money_transfer -app_port 8081 -actors MoneyTransfer,Account node account_server.js

kar run -app money_transfer node account_client.js 

```