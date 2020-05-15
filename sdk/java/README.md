# Java KAR SDK Usage

# Prerequisites
SDK tested using
- Java 1.8.X
- Maven 3.6.X

# Overview
The Java SDK provides:
1.  the class `com.ibm.research.kar.Kar` to communicate with the Kar runtime
2. the package `com.ibm.research.kar.actor` to write and execute actors as part of a Kar component (e.g. a microservice).

## Basic KAR SDK usage
The following code examples show how to use the Kar SDK.

### Invoke a Service:
```java
import javax.json.Json;
import javax.json.JsonObject;
import javax.ws.rs.core.Response;

import com.ibm.research.kar.Kar;

public static void main(String[] args) {
    // get instance of SDK which generates REST client
    Kar kar = new Kar();

    JsonObject params = Json.createObjectBuilder()
				.add("number",42)
				.build();

    // call service
    Response resp = kar.call("MyService", "increment", params);
}
```

### Call an Actor Method:
```java
import javax.json.Json;
import javax.json.JsonObject;
import javax.ws.rs.core.Response;

import com.ibm.research.kar.Kar;

public static void main(String[] args) {
    // get instance of SDK which generates REST client
    Kar kar = new Kar();

    JsonObject params = Json.createObjectBuilder()
				.add("number",42)
				.build();

    // call service
    Response resp = kar.actorCall("ActorType", "ActorID", "remoteMethodName", params);
}
```

## KAR actor runtime
The KAR actor runtime in `com.ibm.research.kar.actor` allows you to:
- Create an actor using an annotated POJO class
- Execute actors as part of your microservice

### Actor Annotations

Actor annotations example:
```java
package com.ibm.research.kar.example.actors;

import javax.json.Json;
import javax.json.JsonObject;
import javax.ws.rs.core.Response;

import com.ibm.research.kar.Kar;
import com.ibm.research.kar.actor.KarSessionListener;
import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Deactivate;
import com.ibm.research.kar.actor.annotations.LockPolicy;
import com.ibm.research.kar.actor.annotations.Remote;

@Actor
public class MyActor  {

    @Activate // optional actor constructor
    public void init() {
        // init code
    }	
    
    // Expose this method to the actor runtime.
    // Calls to this method are synchronized by default
    @Remote 
    public void updateMyState(JsonObject json) {
        // remote code
    }

    // Specify a read policy to allow concurrent access to method
    @Remote(lockPolicy = LockPolicy.READ)  
    public String readMyState() {
        // read-only code
    }
	
    // methods not annotated as @Remote are 
    // not callable by actor runtime
    public void cannotBeInvoked() {
    }

    @Deactivate // optional actor de-constructor
    public void kill() {
    }
}
```
### KAR SessionID
KAR manages a session ID for actor communications as part of the [KAR programming model](https://github.ibm.com/solsa/kar/blob/master/docs/KAR.md). Actors access the sessionId by implementing `KarSessionListener`:
```java
@Actor
public class MyActor implements KarSessionListener {
    private String sessionid;

    // KAR actor runtime will pass the sessionid
    // using this method
    @Override
    public void setSessionId(String sessionId) {
        this.sessionid = sessionId;	
    }

    @Override
    public String getSessionId() {
        return this.sessionid;
    }
}
```
### Build and include the `kar-actor` module as part of a Kar component
 Using Maven, an example `pom.xml` file to include the `kar` and `kar-runtime` modules into a microservice called `actor-server` is:
 ```xml
<?xml version='1.0' encoding='utf-8'?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xsi:schemaLocation="http://maven.apache.org/POM/4.0.0
    http://maven.apache.org/xsd/maven-4.0.0.xsd">

    <modelVersion>4.0.0</modelVersion>

    <groupId>com.ibm.research.kar.example.actors</groupId>
    <artifactId>kar-example-actors</artifactId>
    <version>1.0-SNAPSHOT</version>
    <packaging>pom</packaging>

    <modules>
        <module>path/to/sdk/java/kar</module>
        <module>path/to/sdk/java/kar-actor</module>
        <module>actor-server</module>
    </modules>
</project>
```
The corresponding`pom.xml` in `actor-server` should include the following dependencies:
```xml
<dependency>
    <groupId>com.ibm.research.kar</groupId>
    <artifactId>kar</artifactId>
    <version>1.0-SNAPSHOT</version>
</dependency>
<dependency>
    <groupId>com.ibm.research.kar.actor</groupId>
    <artifactId>kar-actor</artifactId>
    <version>1.0-SNAPSHOT</version>
</dependency>
```
`kar-actor` also requires the following features as part of the runtime. The featureManager section of the `server.xml` for `openliberty` should look like:
```xml
<featureManager>
    <feature>jaxrs-2.1</feature>
    <feature>jsonb-1.0</feature>
    <feature>mpHealth-2.1</feature>
    <feature>mpConfig-1.3</feature>
    <feature>mpRestClient-1.3</feature>
    <feature>beanValidation-2.0</feature>
    <feature>cdi-2.0</feature>
    <feature>concurrent-1.0</feature>
</featureManager>
```
`kar-actor` loads actors at deploy time. Add the actors to the classpath and expose to the actor runtime as context parameters in `web.xml`.  For example, if you have KAR actor types `Dog` and `Cat` which are implemented by `com.example.Actor1` and `com.example.Actor2`, respectively, your `web.xml` would have:
```xml
<context-param>
    <param-name>kar-actor-classes</param-name>
    <param-value>com.example.Actor1, com.example.Actor2</param-value>
</context-param>
<context-param>
    <param-name>kar-actor-types</param-name>
    <param-value>Dog, Cat</param-value>
</context-param>
```

For a complete example, see the [KAR example actor server](https://github.ibm.com/castrop/kar/tree/master/examples/java/actors)