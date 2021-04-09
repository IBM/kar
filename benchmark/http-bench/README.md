# HTTP Benchmark
Timing of end-to-end and request/response HTTP latencies.


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

## On Kubernetes: communicating with the pod

Build the producer and consumer executables.
```
npm install --prod
make docker
```

Launch the server:
```
kubectl apply -f deploy/server.yaml
```

We need to now find out the pod IP address. To do that we need to look at the yaml of the pod:
```
k get pod http-bench-server -o yaml
```

At the end of the output you should look for the `podIP` field:
```
...
  hostIP: 172.18.0.3
  phase: Running
  podIP: 10.244.1.93
  podIPs:
  - ip: 10.244.1.93
  qosClass: BestEffort
  startTime: "2021-04-09T18:20:16Z"
```
In this case the `podIP` is `10.244.1.93`.

Paste the podIP in `client.js` on line 34:
```
    // Replace pod IP with a valid value.
    return `http://10.244.1.93:9000/${route}`
```

Re-create the docker image:
```
make docker
```

Run the client:
```
kubectl apply -f deploy/client.yaml
```

Check the logs of the client job for the measurement results:
```
kubectl logs jobs/http-bench-client -c client
```

## On Kubernetes: communicating with the pod via service

This is a simpler way to communicate with the pod via a service.

In `client.js` comment line 34 and uncomment line 37:
```
return `http://http-bench-server-service.default.svc.cluster.local:9000/${route}`
```

This will allow for the service IP to be automatically discovered. The service will then propagate the message to the pod.

Update the image:
```
make docker
```

Run the client:
```
kubectl apply -f deploy/client.yaml
```

Check the logs of the client job for the measurement results:
```
kubectl logs jobs/http-bench-client -c client
```