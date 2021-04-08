# Kar Benchmark

Timing of KAR end-to-end and request/response latencies.

Build the producer and consumer executables.
```
npm install --prod
```

To run the benchmark, in one window run:
```
kar run -app bench-js -service bench -actors BenchActor node server.js
```

in another window run:
```
kar run -app bench-js node client.js
```

## Using HTTP2 between sidecars and processes

To run the benchmark, in one window run:
```
kar run -h2c -app bench-js -service bench -actors BenchActor node server.js
```

in another window run:
```
kar run -h2c -app bench-js node client.js
```

## Running with Kubernetes

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
kubectl logs jobs/kar-bench-client -c client
```

When finished the output will be of the following form:
```
Average service call duration: 5.98 ms
Average service request duration: 3.41 ms
Average service response duration: 2.74 ms
Average actor call duration: 6.63 ms
Average actor request duration: 3.89 ms
Average actor response duration: 2.68 ms
```

Clean-up:
```
kubectl delete pod bench-server
kubectl delete job kar-bench-client
```