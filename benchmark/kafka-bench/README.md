# Kafka Benchmark
Kafka end-to-end micro benchmarking using the Sarama Kafka library.

This project measures the Kafka end-to-end time and the time to perform the initial request.


Build the producer and consumer executables.
```
make build
```

To run the benchmark, in one window run:
```
$GOPATH/bin/consumer
```

in another window run:
```
$GOPATH/bin/producer
```

## On Kubernetes

First, deploy kind cluster on your machine following the usual instructions [TODO].

Build and push the image:
```
make docker
```

Deploy server:
```
kubectl apply -f deploy/server.yaml
```

Deploy client:
```
kubectl apply -f deploy/client.yaml
```

Check the results:
```
kubectl logs kafka-bench-producer
```
This pod should show something like:
```
2021/08/16 20:48:06.108391 [INFO] Inside Setup!
2021/08/16 20:48:06.108709 [INFO] Sarama return consumer up and running!...
2021/08/16 20:57:02.638099 [INFO] Average Kafka end-to-end time: 5.018634300000017 ms
Message is stored in topic(simple-topic)/partition(0)/offset(16095)
2021/08/16 20:57:03.479145 [INFO] Kafka: end-to-end: samples = 10000; mean = 5.018634300000017; stddev = 2.1366015896192483
```

Checking the other pod:
```
kubectl logs kafka-bench-consumer
```
This pod should show:
```
2021/08/16 20:47:59.327039 [INFO] Starting consumer...
2021/08/16 20:47:59.327160 [INFO] Kafka brokers is [localhost:31093]
2021/08/16 20:47:59.355816 [INFO] Inside Setup!
2021/08/16 20:47:59.355880 [INFO] Sarama consumer up and running!...
2021/08/16 20:57:03.430480 [INFO] Average Kafka request time: 2.747966399999998 ms
```

Clean-up:
```
kubectl delete pod kafka-bench-producer
kubectl delete pod kafka-bench-consumer
```
