TODO: This appears to not work....what is missing?

## Microprofile Open Tracing
Preliminary [Open Tracing](https://opentracing.io/) is enabled for `philosophers`.  

To use when running Jaeger and `philosophers` on localhost:

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

