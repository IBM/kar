# Java Actors example

## Prerequisites
- Java 1.8.x
- Maven 3.6.x

Note: the SDK and example code have been tested using MicroProfile 3.3 and the Open Liberty Plugin 3.2 (which pulls v20.0.0.X of openliberty)

## Building the example

```shell
mvn clean install
```

## Running the example
In one terminal start the server
```shell
cd ./actor-server
kar -app actor -service dummy -actors dummy,dummy2,calculator mvn liberty:run
```

Wait a few seconds until you see
```shell
...
2020/05/15 10:30:58 [STDOUT] [INFO] [AUDIT   ] CWWKT0016I: Web application available (default_host): http://192.168.0.24:8080/health/
2020/05/15 10:30:59 [STDOUT] [INFO] [AUDIT   ] CWWKT0016I: Web application available (default_host): http://192.168.0.24:8080/
2020/05/15 10:30:59 [STDOUT] [INFO] [AUDIT   ] CWWKZ0001I: Application kar-example-actor-server started in 1.391 seconds.
2020/05/15 10:30:59 [STDOUT] [INFO] [AUDIT   ] CWWKF0012I: The server installed the following features: [beanValidation-2.0, cdi-2.0, concurrent-1.0, el-3.0, jaxrs-2.1, jaxrsClient-2.1, jndi-1.0, json-1.0, jsonb-1.0, jsonp-1.1, mpConfig-1.3, mpHealth-2.1, mpRestClient-1.3, servlet-4.0].
2020/05/15 10:30:59 [STDOUT] [INFO] [AUDIT   ] CWWKF0011I: The defaultServer server is ready to run a smarter planet. The defaultServer server started in 4.124 seconds.
```

Then, in a second terminal do
```shell
kar -runtime_port 32123 -app actor curl -s -H "Content-Type: application/kar+json" -X POST http://localhost:32123/kar/v1/actor/dummy/dummyid/call/canBeInvoked -d '[{ "number": 10}]'
```

You should see output like:
```shell
2020/05/15 10:47:09 [STDOUT] {"number":12}
```
## Microprofile Open Tracing
Preliminary [Open Tracing](https://opentracing.io/) is enabled for `actor-server`.  

To use when running Jaeger and `actor-server` on localhost:

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

4. After a period of time, the trace will appear in the Jaeger UI search as service `kar-example-actor-server`


