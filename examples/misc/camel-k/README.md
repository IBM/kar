# Reactive Camel-K example with KAR Kafka using Cloud Events

In this example, Camel-K is used to create a reactive stream to repeatedly request stock price updates from an external service, pack the price as a Cloud Event, send the event to a consumer using KAR's Kafka instance, unpack the cloud event on the consumer side and publish the price to the kar-output Slack channel.

The producer and the consumer are automatically deployed in the cluster by e=invoking `kamel run`.

The Cloud Events dependency is passed to `kamel run` on the command line like so:

```
-d github:cloudevents/sdk-java
```

The `gson` dependency, which is a Camel dependency is passed in like so:

```
-d camel:camel-gson
```

To pass an environment variable to the `kamel run` invocation use:

```
-e MY_ENV_VAR=some-value
```

## Installing Camel K

To run Camel K, the `kamel` executable is needed. The executable used in this example has been built from source to include the latest changes to camel-k trunk which are required for handling the dependencies.

```
git clone git@github.com:apache/camel-k.git
cd camel-k
git checkout 5bb92cf0b8df25787a134bc478620252adddf10f
```

### Option 1: Installing from scratch (advanced)

Make sure your system satisfies the Camel K dependencies detailed here: https://camel.apache.org/camel-k/latest/developers.html

To install Camel K from source some changes are required to the source code to allow the `kamel` image to be pushed to Docker. Replace the `docker.io/apache/camel-k:1.1.0-SNAPSHOT` with `docker.io/<your_own_repo>/camel-k:1.1.0-SNAPSHOT` in the yaml files belonging to the cloned version of Camel K:

```
deploy/olm-catalog/camel-k-dev/1.1.0-snapshot/camel-k.v1.1.0-snapshot.clusterserviceversion.yaml
deploy/operator-deployment.yaml b/deploy/operator-deployment.yaml
script/Makefile
script/images_push.sh
```

Once this change is done run the following commands in the camel-k root directory:

```
make ; make images
```

Push the image to your Docker repository:

```
docker push docker.io/<your_own_repo>/camel-k:1.1.0-SNAPSHOT
```

The image will be pushed to your Docker Hub Repository and used during the `kamel` executable invocation.

### Option 2: Use a pre-existing Camel K image

Replace the `docker.io/apache/camel-k:1.1.0-SNAPSHOT` with `docker.io/doru1004/camel-k:1.1.0-SNAPSHOT` in the yaml files belonging to the cloned version of Camel K:

```
deploy/olm-catalog/camel-k-dev/1.1.0-snapshot/camel-k.v1.1.0-snapshot.clusterserviceversion.yaml
deploy/operator-deployment.yaml b/deploy/operator-deployment.yaml
script/Makefile
script/images_push.sh
```

Then build the `kamel` executable by running:

```
make
```

### Option 3: Use a pre-existing Camel K executable

Camel K is written in Go and self-contained so one can reuse the executable obtained following Options 1 or 2 on another machine. This is the one built by us: https://ibm.box.com/s/meszq51n9reix4yecx6iy60k1e69k5e7

### Final installation step (regardless which option you used)

To install Camel K as part of your kind cluster run the following command:

```
kamel install --registry=registry:5000 --registry-insecure
```

This command assumes that the registry exists. The registry is created when invoking the `start-kind.sh` script.

This will deploy a camel-k operator inside your kind cluster.

## Steps to run the example

Create Kafka topic:

```
sh createTopic.sh
```

For both producer and consumer services, the address of the Kafka broker can be passed using the `KAR_KAFKA_CLUSTER_IP` environment variable. When using the `kind` cluster dev deployment (used in the other KAR examples) use the following to pass the address of the Kafka service: `kar-kafka-0.kar-system`:
```
-e KAR_KAFKA_CLUSTER_IP=kar-kafka-0.kar-system
```

For other setups look up the IP address of the kafka service using the command:

```
kubectl get services --all-namespaces
```

and add the following option to the `kamel run` commands below:

```
-e KAR_KAFKA_CLUSTER_IP=X.X.X.X
```

Below we show the commands for invoking the Prdocer/Consumer example.

Consumer:

```
kamel run -d github:cloudevents/sdk-java src/main/java/org/apache/camel/kamel/KafkaConsumer.java -e SLACK_KAR_OUTPUT_WEBHOOK=<webhook_url> -e KAR_KAFKA_CLUSTER_IP=<kafka_ip>
```

To view the output of the Consumer in the Slack channel of choice pass the `SLACK_KAR_OUTPUT_WEBHOOK` to the Consumer invocation.

Producer

```
kamel run -d camel:camel-gson -d github:cloudevents/sdk-java src/main/java/org/apache/camel/kamel/KafkaProducer.java -e KAR_KAFKA_CLUSTER_IP=<kafka_ip>
```

Optionally add `--dev` to any of the commands to view the log output.
