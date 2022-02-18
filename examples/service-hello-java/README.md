<!--
# Copyright IBM Corporation 2020,2022
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
-->

# Example for using KAR with Java Microprofile 3.X

## Greeting Server Example

This example demonstrates how to use the
Java Microprofile KAR SDK in conjunction with Microprofile based
microservices. The Server provides a greeting service that responds
to requests with a message.

### Prerequisites
- Java 11
- Maven 3.6+

### Building
Build the client and server applications by doing `mvn package`

### Run the Server without KAR and interact via curl:

1. Launch the Server
```shell
sh -c 'export KAR_APP_PORT=8080; mvn liberty:run'
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
(%) kar run -app hello-java java -jar client/target/kar-hello-client-jar-with-dependencies.jar
2020/10/06 09:47:44.068160 [STDOUT] Hello Gandalf the Grey
2020/10/06 09:47:44.068176 [STDOUT] SUCCESS!
```

3. Use the `kar` cli to invoke a route directly (the content type for request bodies defaults to application/json).
```shell
(%) kar rest -app hello-java post greeter helloJson '{"name": "Alan Turing"}'
2020/10/06 09:48:10.929784 [STDOUT] {"greetings":"Hello Alan Turing"}
```
Or invoke the `text/plain` route with an explicit content type:
```shell
(%) kar rest -app hello-java -content_type text/plain post greeter helloText 'Gandalf the Grey'
2020/10/06 09:48:29.644326 [STDOUT] Hello Gandalf the Grey
```

4. If the service endpoint being invoked requires more sophisticated
headers or other features not supported by the `kar rest` command, it
is still possible to use curl. However, the curl command is now using
KAR's REST API to make the service call via a `kar` sidecar.

```shell
(%) kar run -runtime_port 32123 -app hello-java curl -s -X POST -H "Content-Type: text/plain" http://localhost:32123/kar/v1/service/greeter/call/helloText -d 'Gandalf the Grey'
2020/10/06 09:49:45.300122 [STDOUT] Hello Gandalf the Grey
```

## Looking Inside the Server Code

The server code in HelloServices.java uses standard JAX-RS annotations like `@POST`,
`@Path` and `@Consumes` to specify the endpoints provided by the server.

The only KAR-specific detail is the use of the `KAR_APP_PORT` environment variable
in the specification of the `httpPort` in [server.xml](./server/src/main/liberty/config/server.xml).
This enables the `kar` sidecar to control the port that the JVM process
will use to accept incoming requests.
