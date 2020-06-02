# Java KAR SDK Usage

# Prerequisites
SDK tested using
- Java 1.8.X
- Maven 3.6.X

Note: the SDK and example code have been tested using MicroProfile 3.3 and the Open Liberty Plugin 3.2 (which pulls v20.0.0.X of openliberty)

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

import static com.ibm.research.kar.Kar.*;

public static void main(String[] args) {
    JsonObject params = Json.createObjectBuilder()
				.add("number",42)
				.build();

    // call service
    Response resp = call("MyService", "increment", params);
}
```

### Call an Actor Method:
```java
import javax.json.Json;
import javax.json.JsonObject;
import javax.ws.rs.core.Response;

import static com.ibm.research.kar.Kar.*;

public static void main(String[] args) {

    JsonObject params = Json.createObjectBuilder()
				.add("number",42)
				.build();

    // call service
    Response resp = actorCall("ActorType", "ActorID", "remoteMethodName", params);
}
```

## KAR actor runtime
The KAR actor runtime in `com.ibm.research.kar.actor` allows you to:
- Create an actor using an annotated POJO class
- Execute actors as part of your microservice

KAR requires all Actor classes to implement the ActorInstance interface. 
### Actor Instance
```java
public interface ActorInstance extends ActorRef {

  // Allow KAR to get and set session ids   
  public String getSession();
  public void setSession(String session);

  // set actor ID and Type
  public void setType(String type);
  public void setId(String id);
}
```
The ActorInstance includes two methods to manage session IDs, which KAR uses for actor communications as part of the [KAR programming model](https://github.ibm.com/solsa/kar/blob/master/docs/KAR.md).

### Actor Annotations

Actor annotations example:
```java
package com.ibm.research.kar.example.actors;

import javax.json.Json;
import javax.json.JsonObject;
import javax.ws.rs.core.Response;

import com.ibm.research.kar.actor.KarSessionListener;
import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Deactivate;
import com.ibm.research.kar.actor.annotations.LockPolicy;
import com.ibm.research.kar.actor.annotations.Remote;

@Actor
public class MyActor implements ActorInstance {

    @Activate // optional actor constructor
    public void init() {
        // init code
    }	
    
    // Expose this method to the actor runtime.
    // KAR synchronizes requests to the actor by default
    @Remote
    public void updateMyState(JsonObject json) {
        // remote code
    }

    
    // Expose this method to the actor runtime.
    // KAR synchronizes requests to the actor by default
    @Remote 
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

    //.... ActorInstance implementation would be below
    //.....
}
```

### Build and include the `kar` module as part of a Kar component
 Using Maven, an example `pom.xml` file to include the `kar` module into a microservice called `actor-server` is:
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
        <module>actor-server</module>
    </modules>
</project>
```
The corresponding`pom.xml` in `actor-server` should include the following dependency:
```xml
<dependency>
    <groupId>com.ibm.research.kar</groupId>
    <artifactId>kar</artifactId>
    <version>1.0-SNAPSHOT</version>
</dependency>
```
`kar` requires the following features as part of the runtime. The featureManager section of the `server.xml` for `openliberty` should look like:
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
`kar` loads actors at deploy time. Add the actors to the classpath and expose to the actor runtime as context parameters in `web.xml`.  For example, if you have KAR actor types `Dog` and `Cat` which are implemented by `com.example.Actor1` and `com.example.Actor2`, respectively, your `web.xml` would have:
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