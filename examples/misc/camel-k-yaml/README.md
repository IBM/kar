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

# Camel K example with KAR Kafka and Cloud Events using Yaml to specify integrations

This example builds on the Camel K example. Camel K uses reactive streams to repeatedly request stock price updates from an external service, pack the price as a Cloud Event, send the event to a consumer using KAR's Kafka instance, unpack the cloud event on the consumer side and publish the price to the kar-output Slack channel.

This example uses YAML to specify the integration code.

Note: unlike the Camel K example, this example does not use the `choice()` Camel DSL primitive, instead it uses the `filter()` primitive.

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

Run the subscriber:
```
sh run-subscriber.sh
```

Run the publisher:
```
sh run-publisher.sh
```
