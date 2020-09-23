# Example for using KAR with Java Microprofile 3.X

## Greeting Server Example

This example demonstrates how to use the
Java Microprofile KAR SDK in conjunction with Microprofile based
microservices.

The Server provides a greeting service that responds to requests with
a message.  The Client acts as a proxy to the Server (just to
demonstrate KAR, obviously you can directly invoke the Server if you
want to).  The Client uses the `kar-rest-client` library to call the
Server.

### Prerequisites
- Java 11
- Maven 3.6+

### Building
Build the client and server applications by doing `mvn package`

### Run the Server without KAR and interact via curl:

1. Launch the Server
```shell
mvn liberty:run
```
2. Invoke routes using curl
```shell
(%) curl -s -X POST -H "Content-Type: text/plain" http://localhost:8080/helloText -d 'Gandalf the Grey'
Hello Gandalf the Grey
```
```shell
(%) curl -s -X POST -H "Content-Type: application/json" http://localhost:8080/helloJson -d '{"name": "Alan Turing"}'
{"greetings":"Hello Alan Turing"}
```

### Run using KAR

1. Launch the Server
```shell
kar run -app hello-java -service greeter mvn liberty:run
```

2. Run a test client
```shell
kar run -app hello-java java -jar target/kar-hello-client-jar-with-dependencies.jar
```

3. You can still use curl to invoke the service via kar, but you will now KAR's service invocation route
```shell
kar run -runtime_port 32123 -app hello-java curl -s -X POST -H "Content-Type: text/plain" http://localhost:32123/kar/v1/service/greeter/call/helloText -d 'Gandalf the Grey'
```
