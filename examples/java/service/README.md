# Example for using KAR with Java Microprofile 3.X

## Increment Server Example
This example demonstrates how to use the Java Microprofile KAR SDK to make synchronous and asynchronous calls between microservices. The Server provides a number service that increments numbers.  The Client acts as a proxy to the Server (just to demonstrate KAR, obviously you can directly invoke the Server if you want to).  The Client uses the `kar-rest-client` library to call the Server.

### Prerequisites
- Apache Maven 3.6.3 or above
- Java 1.8.0 or above

### Building
Inside `example/java/service` run `mvn install`.  This will build the Client and Server as well as `kar-rest-client` and manage the packaging.

### Running Example Incr Code wit Local Installation of KAR
Assuming you have [KAR installed](https://github.ibm.com/solsa/kar/blob/master/docs/getting-started.md):

#### 1. Launch Number Service
Inside the `service/server` directory:
`kar run -app example -service number mvn liberty:run`

#### 2. Launch Client Service
Inside the `service/client` directory:
`kar run -app_port 9090 -app example -service client mvn liberty:run`

#### 3. Invoke the Client using curl
Inside the `service/scripts` directory run the script that contains the curl request:
`$ ./clientPost.sh`

If all goes well you should get a response `11`.
