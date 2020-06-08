# Camel example with KAR Kafka using Cloud Events

This patch introduces a new example which uses Apache Camel as an integration framework for adding event sources.

The event source exercised in this example is the console input stream. The example turns user messages into Cloud Events, published via KAR's Kafka instance on the `HelloEvent` topic.


## Steps to run the example

Install Cloud Events Java SDK: 

```
git clone git@github.com:cloudevents/sdk-java.git
cd sdk-java
mvn install
```

Create topics:
```
cd examples/camel
sh createTopics.sh
mvn compile
```

Run the consumer:
```
cd examples/camel
mvn compile exec:java -Pkafka-consumer
```

Run the producer:
```
cd examples/camel
mvn compile exec:java -Pkafka-producer
```
