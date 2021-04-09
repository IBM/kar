# Kafka Benchmark
Kafka end-to-end micro benchmarking using the Sarama Kafka library.

This project measures the Kafka end-to-end time and the time to perform the initial request.


Build the producer and consumer executables.
```
make build
```

To run the benchmark, in one window run:
```
./consumer/consumer
```

in another window run:
```
./producer/producer
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
time="2021-04-09T00:34:46Z" level=info msg="Average Kafka end-to-end time: 3.2248899082568805 ms" source="main.go:147"
```

Checking the other pod:
```
kubectl logs kafka-bench-consumer
```
This pod should show:
```
time="2021-04-09T00:34:46Z" level=info msg="Average Kafka request time: 1.5344495412844037 ms" source="main.go:159"
```

Clean-up:
```
kubectl delete pod kafka-bench-producer
kubectl delete pod kafka-bench-consumer
```
