# Java Actors example

## Prerequisites
- Java 11
- Maven 3.6+

Note: the SDK and example code have been tested using MicroProfile 3.3 and the Open Liberty Plugin 3.2 (which pulls v20.0.0.X of openliberty)

## Building the example

In `examples/java/actors`:
```shell
mvn clean install
```

## Running the example
In one terminal start the server
```shell
kar run -app example -actors sample,calculator mvn liberty:run
```

Wait a few seconds until you see something similar to:
```shell
...
2020/06/17 14:17:57.219810 [STDOUT] [INFO] [AUDIT   ] CWWKF0012I: The server installed the following features: [beanValidation-2.0, cdi-2.0, concurrent-1.0, el-3.0, jaxrs-2.1, jaxrsClient-2.1, jndi-1.0, json-1.0, jsonb-1.0, jsonp-1.1, mpConfig-1.3, mpHealth-2.1, mpOpenTracing-1.3, mpRestClient-1.3, opentracing-1.3, servlet-4.0].
2020/06/17 14:17:57.221198 [STDOUT] [INFO] [AUDIT   ] CWWKF0011I: The defaultServer server is ready to run a smarter planet. The defaultServer server started in 14.519 seconds.
```

### Use kar
You can use the `kar` cli to invoke an actor method
```shell
kar invoke -app example sample abc canBeInvoked '{"number":10}'
```

You should see output like:
```shell
2020/06/26 11:09:36.397079 [STDOUT] {"value":{"number":11}}
```

#### Use kar+curl (lower-level)
You can use the `kar` cli to wrap a curl command that directly invokes the lower-level KAR REST API:
```shell
kar run -runtime_port 32123 -app example curl -s -H "Content-Type: application/kar+json" -X POST http://localhost:32123/kar/v1/actor/sample/abc/call/canBeInvoked -d '[{ "number": 10}]'
```

You should see output like:
```shell
2020/05/15 10:47:09 [STDOUT] {"value":{"number":11}}
```

#### Use Java example
You can run a simple test Java application packaged with `kar-rest-client` that uses the KAR Java SDK to call an actor:

```shell
$ ../../../sdk-java/kar-rest-client/scripts/runactor.sh
```
You should see output like:
```shell
2020/06/17 15:02:09.032753 [STDOUT] {"number":43}
```

## Microprofile Open Tracing
Preliminary [Open Tracing](https://opentracing.io/) is enabled for `kar-actor-example`.  

To use when running Jaeger and `kar-actor-example` on localhost:

1. Run Jaeger backend:
```
$ docker run -d --name jaeger \
  -e COLLECTOR_ZIPKIN_HTTP_PORT=9411 \
  -p 5775:5775/udp \
  -p 6831:6831/udp \
  -p 6832:6832/udp \
  -p 5778:5778 \
  -p 16686:16686 \
  -p 14268:14268 \
  -p 14250:14250 \
  -p 9411:9411 \
  jaegertracing/all-in-one:1.18
  ```

2. Browse to  [`http://localhost:16686/`](http://localhost:16686/) to access the Jaeger UI

3. Run the example.

4. After a period of time, the trace will appear in the Jaeger UI search.

