# HTTP Benchmark
Timing of end-to-end and request/response HTTP latencies.

## Using local processes

Build the producer and consumer executables.
```
npm install --prod
```

To run the benchmark, in one window run:
```
node server.js
```

in another window run:
```
node client.js
```

## On Kubernetes

Launch the server:
```
kubectl apply -f deploy/server.yaml
```

Run the client:
```
kubectl apply -f deploy/client.yaml
```

Check the logs of the client job for the measurement results:
```
kubectl logs jobs/http-bench-client -c client
```
