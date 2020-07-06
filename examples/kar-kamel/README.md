# Camel K example with KAR Processor, KAR Kafka and Cloud Events using Yaml to specify integrations

This example builds on the Camel K example. The example is split into three components: two generic source and sink components and one user defined components.

The source component uses Camel K's reactive streams to repeatedly request stock price updates from an external service. The stock information is packed as a Cloud Event and forwarded to a user implemented processor launched using KAR. The cloud event is published on the `InputStockEvent` Kafka topic.

The user-defined processor unpacks the cloud event, adds it to the list of existing stock prices, computes the maximum stock price seen so far and then assembles a Cloud Event which contains the output string to be printed in Slack. The cloud event is published on the `OutputStockEvent` Kafka topic.

The sink component receives the Cloud Event containing the output message and forward the message to the Slack kar-output channel.

This example uses YAML to specify the generic integration code which is used to interact with the event source (stock price service) and the sink (Slack).

The KAR component is implemented in Javascript.

This example relies on access to a `kamel` executable. Follow the steps in the `camel-k` example for how to satisfy this dependency.

## Steps to run the example

For the consumer to output to a Slack channel, expose the incoming webhook address via the `SLACK_KAR_OUTPUT_WEBHOOK` environment variable. If the variable is not set the output will be emitted only as a log message.

Export the IP address of KAR's Kafka service via the `KAR_KAFKA_CLUSTER_IP` environemnt variable.

Move to the example folder:

```
cd examples/kar-kamel
```

Create topics for this example:

```
sh createTopics.sh
```

Since the KAR user-defined component is implemented in Javascript, run `npm`:

```
npm install
```

## Running the components

Run the sink component:
```
sh run-sink.sh
```

Run the user-defined component:
```
kar -app stock-price-manager -runtime_port 3502 -app_port 8082 -service stock-manager -actors StockManager -- node kar-server.js
```
For additional output details pass `-v info` to the kar CLI.

Run the source component:
```
sh run-source.sh
```
