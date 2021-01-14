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

# KAR, Camel, and CloudEvents

This example demonstrates how to build KAR solutions that consume and produce
external events or data streams.

## Apache Camel

KAR components can natively produce and consume events from Kafka topics using
simple APIs built into KAR. KAR leverages [Apache
Camel](https://camel.apache.org) to access hundreds of data services beyond
Kafka. KAR makes it easy to configure and run Camel integrations that connect
external data services to Kafka topics. These integrations are either sources or
sinks. Sources fetch or accept data from external services that they feed to
Kafka topics. Sinks forward messages from Kafka topics to external services.

## YAML integration language

Sources and sinks may be implemented using any integration language supported by
the [Camel-K project](https://camel.apache.org/camel-k/latest/index.html).
However, we recommend using the [YAML integration
language](https://camel.apache.org/camel-k/latest/languages/yaml.html) as
illustrated in this example. YAML makes it easy to configure Camel integrations
without any coding.

## CloudEvents

KAR leverages [CloudEvents](https://cloudevents.io) to encode events in a
portable, cloud-native way. KAR facilitates the construction and deconstruction
of CloudEvents in Camel sources and sinks.

## Dependencies

Camel integrations run on a Java Virtual Machine. KAR leverages the `kamel` CLI
from the [Camel-K project](https://camel.apache.org/camel-k/latest/index.html)
and [Apache Maven](https://maven.apache.org) to assemble the artifacts required
to run a Camel integration (essentially a collection of jar files).

In contrast to Camel-K today, KAR does not require a Kubernetes cluster to run
integrations. Moreover KAR does not require the Camel-K operator to deploy
integrations to Kubernetes.

## Example description

This example application combines three components to analyze stock prices:
- A Camel source periodically fetches stock prices from a web service and feeds
  them to a Kafka topic as CloudEvents.
- A KAR component subscribes to this topic and analyses trends, publishing
  insights to a second Kafka topic as CloudEvents.
- A Camel sink posts these insights to a Slack Channel named `kar-output`.

## Slack Webhook

This example assumes a [webhook URL](https://api.slack.com/messaging/webhooks)
for the Slack channel is provided via the environment variable `SLACK_WEBHOOK`.

## Example code

The Camel source is implemented in file [input.yaml](input.yaml). The Camel sink
is implemented in file [output.yaml](output.yaml). The KAR component is
implemented in JavaScript in file [processor.js](processor.js). A Kubernetes
deployment template is provided in [stocks-dev.yaml](deploy/stocks-dev.yaml).

## Build and run locally

To prepare the KAR component for execution run:
```
npm install --prod
```

To launch the KAR component run:
```
kar run -app stocks -actors StockManager -- node processor.js
```
This KAR component will create the necessary Kafka topics.

To launch the source run:
```
../../scripts/kamel-local-run.sh input.yaml
```

To launch the sink run:
```
../../scripts/kamel-local-run.sh output.yaml
```

## Build and run inside a container

Building the user part of the example:
```
docker build . -t stock-processor
```

Launching and running the user part of the example:
```
../../scripts/kar-docker-run.sh -app stocks -actors StockManager stock-processor
```

Building, launching and running the example source and sink parts can be done in two ways:
(1) using docker directly
(2) using kamel

### Launching sources and sink using Docker

To build container images for the three components run:
```
docker build workspace-http-source -t stock-source
docker build workspace-slack-sink -t stock-sink
```

To launch the example run:
```

docker run --network kar-bus stock-source --detach
docker run --network kar-bus --env SLACK_WEBHOOK=$SLACK_WEBHOOK stock-sink --detach
```

### Launching sources and sink using Kamel

Ensure Docker is available. Identify the local docker repository as something like:
```
export LOCAL_DOCKER_REGISTRY=docker.io/<registry-name>
```

For this part of the example KAFKA_BROKERS needs to be set to:
```
export KAFKA_BROKERS=kafka:9092
```

Build base image to be used as a starting point for all the integration images.
```
kamel local create --base-image --container-registry ${LOCAL_DOCKER_REGISTRY}
```
This step will be performed by kamel if no base image is found.

Launch the example:
```
../../scripts/kamel-docker-run.sh --image ${LOCAL_DOCKER_REGISTRY}/stock-source-image input.yaml
../../scripts/kamel-docker-run.sh --image ${LOCAL_DOCKER_REGISTRY}/stock-sink-image output.yaml
```

## Build and run using Kind development cluster

To build container images for the three components run:
```
docker build . -t localhost:5000/examples/stock-processor
docker build workspace-http-source -t localhost:5000/examples/stock-source
docker build workspace-slack-sink -t localhost:5000/examples/stock-sink
```

To push these container images to the local registry run:
```
docker push localhost:5000/examples/stock-processor
docker push localhost:5000/examples/stock-source
docker push localhost:5000/examples/stock-sink
```

To deploy the example to Kubernetes, we first need to create a Kubernetes secret
containing the Slack webhook. Run:
```
kubectl create secret generic slack --from-literal=webhook=$SLACK_WEBHOOK
```

To deploy the example run:
```
kubectl apply -f deploy/stocks-dev.yaml
```

To undeploy the example run:
```
kubectl delete -f deploy/stocks-dev.yaml
```
