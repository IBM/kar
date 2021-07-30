<!--
# Copyright IBM Corporation 2020,2021
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

# Java KAR SDK Usage

# Prerequisites
- Java 11
- Maven 3.6+

# Overview

The Java SDK provides an implementation of the KAR programming model
that utilizes familar Java frameworks such as javax.json and javax.ws.
To be complete, the core KAR Java SDK must be embedded in a Java
middleware framework that provides it with webserver capabilities.
We have a completed implementation based on Open Liberty and are
developing a second one based on Quarkus.

The Java SDK is structured internally into three sub-modules:
1. `kar-runtime-core` - Defines the core abstractions of the
    programming model and the runtime system that implements them
    on top of an abstract REST client that represents the KAR
    service mesh.
2. `kar-runtime-liberty` - An implementation of the abstract REST
   client using Open Liberty as the underlying server framework.
3. `kar-runtime-quarkus` - An implementation of the abstract REST
   client using Quarkus as the underlying server framework. This is
   still a work in progress.

To use the Java SDK in an application component, you declare a maven
dependency on one of `kar-runtime-liberty` or `kar-runtime-quarkus` as
shown in more detail below.  You then follow the framework-specific
instructions on using annotations or xml configuration files
toconfigure your component (eg. by specifying the Actor types).


# Building

Builds are driven by maven.  The basic commmand is `mvn install`.

# Basic KAR SDK usage

The developer-facing APIs for the KAR SDK are all defined in
`kar-runtime-core`.  The primary API is defined by
`com.ibm.research.kar.Kar` and is supported by types and annotations
in `com.ibm.research.kar.actor` and its sub-packages. The package
`com.ibm.research.kar.runtime` defines internal APIs that are not
intended for developer use; they are only made public to enable them
to be invoked from support code within `kar-runtime-liberty` and
`kar-runtime-quarkus`.

The following code examples show how to use the Kar SDK.

## Invoke a Service:
```java
import javax.json.Json;
import javax.json.JsonObject;
import javax.json.JsonValue;

import static com.ibm.research.kar.Kar.*;

public static void main(String[] args) {
    JsonObject params = Json.createObjectBuilder()
				.add("number",42)
				.build();

    // call service
    JsonValue value = call("MyService", "increment", params);
}
```

## Call an Actor Method:
```java
import javax.json.Json;
import javax.json.JsonObject;
import javax.json.JsonValue;

import static com.ibm.research.kar.Kar.*;

public static void main(String[] args) {

    JsonObject params = Json.createObjectBuilder()
				.add("number",42)
				.build();

    // call service
    JsonValue value = actorCall("ActorType", "ActorID", "remoteMethodName", params);
}
```

## Invoke a service asynchronously
```java
import javax.json.Json;
import javax.json.JsonObject;
import javax.json.JsonValue;

import static com.ibm.research.kar.Kar.*;

public static void main(String[] args) {

    JsonObject params = Json.createObjectBuilder()
				.add("number",42)
				.build();

    // call service asynchronously
   CompletionStage<JsonValue> cf = callAsync("MyService", "increment", params);

   JsonValue value = cf
                    .toCompletableFuture()
                    .get();
}
```

## Call an Actor Method asynchronously
```java
import javax.json.Json;
import javax.json.JsonObject;
import javax.json.JsonValue;

import static com.ibm.research.kar.Kar.*;

public static void main(String[] args) {

    JsonObject params = Json.createObjectBuilder()
				.add("number",42)
				.build();
    // call actor asnchronously
    CompletionStage<JsonValue> cf = actorCallAsync("ActorType", "ActorID", "remoteMethodName", params);

    JsonValue value = cf
                    .toCompletableFuture()
                    .get();
}
```

## KAR actors

The KAR actor runtime allows you to:
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
The ActorInstance includes two methods to manage session IDs, which KAR uses for actor communications as part of the [KAR programming model](../docs/KAR.md).

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


# Using the Open Liberty based KAR SDK

Note1: the SDK and example code have been tested using MicroProfile 3.3 and the Open Liberty Plugin 3.2 (which pulls v20.0.0.X of openliberty). You should not use v20.0.0.11 because of a known bug in the Microprofile Rest Client.

In addition to writing the framework independent application code
using the KAR SDK, you will need to write some additional bits of
boilerplate to enable Open Liberty to execute your component.

1. You will need to add a stanza to your `pom.xml` to declare a
   dependency on `kar-runtime-liberty` and on some Open Liberty
   dependencies used within KAR.
2. You will need to provide a class that extends
   `javax.ws.rs.core.Application`.
3. If your application component contains any KAR Actor types, you
   will need to specify them in your `web.xml`.

Using Maven, an example `pom.xml` to include `kar-runtime-liberty` module into a microservice called `kar-actor-example` is:
 ```xml
<?xml version='1.0' encoding='utf-8'?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xsi:schemaLocation="http://maven.apache.org/POM/4.0.0
    http://maven.apache.org/xsd/maven-4.0.0.xsd">

    <modelVersion>4.0.0</modelVersion>

    <groupId>com.ibm.research.kar.example.actors</groupId>
    <artifactId>kar-example-actors</artifactId>
    <version>1.0.0</version>
    <packaging>pom</packaging>

    <modules>
        <!-- Your application -->
        <module>kar-actor-example</module>
    </modules>
</project>
```
The corresponding`pom.xml` in `kar-actor-example` should include the following dependency:
```xml
<!-- KAR SDK -->
<dependency>
	<groupId>com.ibm.research.kar</groupId>
	<artifactId>kar-runtime-liberty</artifactId>
	<version>X.Y.Z</version>
</dependency>
```
`kar-runtime-liberty` requires the following features as part of the runtime. The featureManager section of the `server.xml` for `openliberty` should look like:
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
	<feature>mpOpenTracing-1.3</feature>
</featureManager>
```
`kar-runtime-liberty` loads actors at deploy time. Actor classfiles should be added to your CLASSPATH.  Declare your actors to `kar-runtime-liberty` as context parameters in `web.xml`.  For example, if you have KAR actor types `Dog` and `Cat` which are implemented by `com.example.Actor1` and `com.example.Actor2`, respectively, your `web.xml` would have:
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

# Using the Quarkus based KAR SDK

In addition to writing the framework independent application code
using the KAR SDK, you will need to write some additional bits of
boilerplate to enable Quarkus to execute your component.

TODO:  Once this is working...we need to write the instructions!