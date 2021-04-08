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
