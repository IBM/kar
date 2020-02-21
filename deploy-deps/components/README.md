SolSA solution to deploy dev-mode KAR runtime services onto a cluster.

### Deploying the KAR runtime

Execute the command: `solsa yaml kar-dev.js | kubectl apply -f -`

Components deployed:
1. non-HA Kafka cluster
2. Kafka console pod (to enable debug via kafka-cli tools).

### Debugging Kafka via the kar-kafka-console

1. Connect to the pod: `kubectl exec -it kar-kafka-console-6b984657f-nr48z /bin/bash`

2. Within the pod, you have access to the full set of kakfa cli tools (in `/opt/kafka/bin`).
Use `kar-kafka-0.kar-kafka:9092` as the value for either the `--bootstrap-server` or `--broker-list`
command line argument (depending on which script you are using).
