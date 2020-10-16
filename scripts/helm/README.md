A Helm Chart to deploy a dev-mode KAR runtime onto a cluster.

For detailed instructions, see [getting-started.md](../docs/getting-started.md).

### Components deployed

1. Core KAR
   - Sidecar injection machinery (MutatingWebHook)
   - Secrets containing runtime configuration
2. Supporting Components
   - Redis
   - Kafka (and Zookeeper)

KAR can also be configured to use external Kafka and/or Redis instances by
overriding the default settings from `values.yaml`. For example, to use and
external Kafka set `kafka.internal` to `false` and provide all of the values in
the `kafka.externalConfig` structure.
