# Setting up KAR in a few easy steps

This is a quick summary of the steps required to get up and running with KAR.

1. Follow the list of prerequisites included [here](../../docs/getting-started.md).

2. Clone KAR:

```script
git clone git@github.ibm.com:solsa/kar.git
cd kar
```

3. Build your docker images and push them to kind's internal docker registry:

```shell
make kindPushDev
```

4. Deploy KAR in dev mode by doing:
```shell
./scripts/kar-deploy.sh -dev
```

5. KAR-enable the default namespace:
```shell
./scripts/kar-enable-namespace.sh default
```

6. When you're done or just want to reset your KAR deployment, undeploy KAR in dev mode by doing:
```shell
./scripts/kar-undeploy.sh
```

# Stock price application overview

The three coponents are:
- `stock-client.js` the CLIENT which initiates the request for the latest stock price.
    (1) if a `-s <STOCK_NAME>` is passed when invoking the client, the latest opening value of that stock will be printed.
    (2) if no `-s` flag is provided, the client will attempt to automatically create a portfolio made up of some pre-defined technology stocks. The client will create and publish a purchase event (of type CloudEvent) for each stock.
- `stock-event-sender.js` the intermediate process, EVENT EMITTER, does the following:
    (1) receives the POST request from CLIENT containing the stock identifier;
    (2) requests the stock prices from the third-party API;
    (3) creates a historical prices printing event (of type CloudEvent) on the `historical-prices` topic and publishes it.
- `stock-server.js` the SERVER performs the following functions:
    (1) subscribes to the `historical-prices` topic and then processes any event emitted by the EVENT EMITTER side on that topic. The event is handled by the `cloudevents-sdk` method. If the event has been formed and transmitted correctly, it should be accepted as a CloudEvent by the `StructuredHTTPReceiver.parse()` method.
    (2) subscribes to the `buy-stock` topic and sets the `buy` method of the `Portfolio` actor to handle the incoming events on this topic. The `buy` method records the new purchase in the list of already purchased shares.

# Running the application components locally

As long as the KAR instance is running, the different components of the application can be started and monitored individually.

First, set up the environment:

```shell
source ./scripts/kar-kind-env.sh
cd examples/stockPriceEvents
```

We can now start each component of the application individually. You can do this in three different terminals to better observe the logging information that is emitted by each component (via the `-v info` option).

SERVER:
```shell
kar -v info -app stocks -runtime_port 3502 -app_port 8082 -service price-printer -actors Portfolio -- node stock-server.js
```

EVENT EMITTER:
```shell
kar -v info -app stocks --runtime_port 3501 -app_port 8081 -service price-sender node stock-event-sender.js
```

CLIENT (get stock price):
```shell
kar -v info -app stocks -runtime_port 3503 -app_port 8083 -service stock-client -- node stock-client.js -s GOOG
```
or

CLIENT (create portfolio):
```shell
kar -v info -app stocks -runtime_port 3503 -app_port 8083 -service stock-client -- node stock-client.js
```

Notes:
- The CLIENT can be invoked with the `-e` option for which the default value is `AAPL`.
- `-runtime_port` and `-app_port` ports must differ since all three processes run on the same instance (i.e. your local machine). This is in contrast with running in Kubernetes where each component has its own pod.
- The KAR runtime will ensure the requests are delivered even when components are offline. The requests sent to a component that is offline are delivered when the comnponent comes back online.
- Components belonging to the same application must share the same `-app` name. Within an app, each service must have a unique name given by the `-service` option.
- To stop KAR from delivering a message undeploy and then redeploy KAR (run steps 6 and 4 above).

# Running the application components with Kubernetes

The components can be deployed using Kubernetes.

```shell
cd examples/stockPriceEvents
```

Deploy the three application components:
```shell
kubectl apply -f deploy/client-dev.yaml
kubectl apply -f deploy/event-sender-dev.yaml
kubectl apply -f deploy/server-dev.yaml
```
First, initiate the request for the stock option price by running:

```shell
kubectl logs jobs/stock-price-client -c stock-client
```

Inspect the logs of the other components by running:

```shell
kubectl logs jobs/stock-price-client -c stock-event-sender
kubectl logs stock-price-server -c stock-server
```

When you're done, undeploy the components:
```shell
kubectl delete -f deploy/client-dev.yaml
kubectl delete -f deploy/server-dev.yaml
kubectl delete -f deploy/event-sender-dev.yaml
```
