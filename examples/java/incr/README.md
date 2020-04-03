# Example for using KAR with Java Microprofile 3.X

## Increment Server Example
This example demonstrates how to use the Java Microprofile KAR SDK to make synchronous and asynchronous calls between microservices. 

### Prerequisites
- Apache Maven 3.6.3 or above
- Java 1.8.0 or above

### Running Example Incr Code Locally Without KAR
You can build and run the code locally to see how it works without using KAR. To run, do `mvn liberty:run` or `mvn liberty:dev` in either the client or server directories. Currently, the pom.xml is configured so the example uses Open Liberty v 20.0.0.3 as the runtime.

The server code will install the Number Server and bind to port 9080.  The client code will install the Client Service that calls the Number Server.  The client service will bind to port 9090. You can invoke the Client Service using its REST API, which will in turn cause the Client Service to invoke the Number Server.

The Number Server does the following:
- perform a compute intensive task (increment an integer) on a POST, 
- return an integer on a GET. 

The Client Service will proxy calls to the Number Server using the KAR SDK.
Calls to the Client Service can wait for a response (using path `number/incrSync`), or invoke an asynchronous call (using path `number/aSync`).  Currently, the asynchronous call is mocked and will always return an HTTP status code of OK. 

You can test the server or client using the `serverPost.sh` or `clientPost.sh` scripts in the scripts directory.

### Running the Example Incr Code in Kubernetes with KAR
To run this example in KAR, first install KAR into your Kubernetes cluster following the [Getting Started with KAR](https://github.ibm.com/solsa/kar/blob/master/docs/getting-started.md) guide.  Then:

(We assume we're using the `kar-apps` namespace described in the guide)

1. Deploy this example using `kubectl -n kar-apps -f apply deploy/kubernetes.yaml`. This will use pre-built images for the server `paulccastro/kar-example-server` and `paulcccastro/kar-example/client` on Dockerhub.  Alternately, you can build your own images from the source.

2. Test the installation by connecting to the `kar-example-client` container and running `clientPost.sh` available in the `scripts` directory.  Use `kubectl -n kar-apps exec -it -c kar-example-client <client pod name> sh` to connect to the container. The just cut & paste the contents of `clientPost.sh` into the resulting shell.


### Configuring
You can change the port of the server or client by modifying their respective pom.xml.  If you modify the server port, you will also need to change URI property in the microprofile settings at client/src/main/webapp/META-INF/microprofile.properties. The client `microprofile.properties` file contains a ConfigProperty `useKar`.  When true, the client code will use the KAR SDK.  When false, the client code will bypass KAR and call the Number server directly.

Alternatively, you can directly update the server.xml files.  

Note that KAR has default values where it expects an application to be running.  You can ovverride the defaults using the kar annotations in the deployment YAML.  For example, the following names the app as `incr-example` with service `client` running on port `9090`:

```
        kar.ibm.com/app: incr-example
        kar.ibm.com/service: client
        kar.ibm.com/sendPort: "9090"
```


### KAR for Java Microprofile
This example is designed for use with Java Microprofile 3.X. 

To use the KAR SDK in your own Microprofile application:
- use a runtime that supports Microprofile  3.X or above (currently we've only tested Open Liberty)
- add the KAR SDK to your Classpath
- add the require Microprofile features to `<featureManager>` in `server.xml` 

```
<feature>jaxrs-2.1</feature>
<feature>jsonp-1.1</feature>
<feature>cdi-2.0</feature>
<feature>mpConfig-1.3</feature>
```
