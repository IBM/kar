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
