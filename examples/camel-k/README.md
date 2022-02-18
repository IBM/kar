<!--
# Copyright IBM Corporation 2020,2022
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
from the [Camel-K project](https://camel.apache.org/camel-k/latest/index.html).
At this time, we require a `kamel` CLI built from
[source](https://github.com/apache/camel-k) as the 1.3.0 release is missing critical updates.
```
git clone https://github.com/apache/camel-k
cd camel-k
git checkout 4e0cb8
make build-kamel
```

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
deployment template is provided in [stocks.yaml](deploy/stocks.yaml).

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
kamel local run CloudEventProcessor.java input.yaml
```
To launch the sink, make sure to export the SLACK_WEBHOOK and run:
```
kamel local run CloudEventProcessor.java output.yaml
```

## Prepare container images and deploy to Kubernetes

To deploy the example to Kubernetes, we first need to create a Kubernetes secret
containing the Slack webhook. Run:
```
kubectl create secret generic slack --from-literal=webhook=$SLACK_WEBHOOK
```
To build and push container images for this example to the docker registry on `localhost:5000` run:
```
make dockerDev
```
To deploy the example to a development Kubernetes cluster run:
```
kubectl apply -f deploy/stocks.yaml
```
To undeploy the example run:
```
kubectl delete -f deploy/stocks.yaml
```
