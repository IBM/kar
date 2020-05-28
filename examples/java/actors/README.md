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
kar -runtime_port 32123 -app actor curl -s -H "Content-Type: application/json" -X POST http://localhost:32123/kar/v1/actor/dummy/dummyid/call/canBeInvoked -d '[{ "number": 10}]'
```

You should see output like:
```shell
2020/05/15 10:47:09 [STDOUT] {"number":12}
```
