A Helm Chart to deploy a dev-mode KAR runtime onto a cluster.

For detailed instructions, see [getting-started.md](../docs/getting-started.md).

### Components deployed

1. Core KAR
   - Sidecar injection machinery (MutatingWebHook)
   - Secrets containing runtime configuration
2. Supporting Components
   - Redis
   - Kafka (and Zookeeper)
   - Kafka console pod (to enable debug via Kafka's cli tools).

KAR can also be configured to use external Kafka and/or Redis instances by
overriding the default settings from `values.yaml`. For example, to use and
external Kafka set `kafka.internal` to `false` and provide all of the values in
the `kafka.externalConfig` structure.

### Debugging Kafka via the kar-kafka-console

1. Connect to the pod: `kubectl exec -it kar-kafka-console-6b984657f-nr48z /bin/bash`

2. Within the pod, you have access to the full set of Kafka cli tools (in
   `/opt/kafka/bin`). The environment variables `KAFKA_BOOTSTRAP_SERVER` and
   `KAFKA_BROKER` are available to help you connect to the KAR runtime's
   instance of kafka. For example,
```
bash-4.4# kafka-topics.sh --bootstrap-server $KAFKA_BOOTSTRAP_SERVER --create --topic myTest --partitions 3 --replication-factor 1
bash-4.4# kafka-topics.sh --bootstrap-server $KAFKA_BOOTSTRAP_SERVER --list
myTest
bash-4.4#
```
