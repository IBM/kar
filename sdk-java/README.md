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

# KAR Java SDK Usage

# Prerequisites
- Java 11
- Maven 3.6+

# Overview

The KAR Java SDK provides an implementation of the KAR programming model
that utilizes familar Java frameworks such as javax.json.
To be complete, the KAR Java SDK must be embedded in a Java
middleware framework that provides it with webserver capabilities.

We have two complete implementations of the Java SDK: one based on
Open Liberty that supports a more traditional imperative programming style and
one based on Quarkus that supports a reactive programming style.
You should pick the appropriate KAR Java SDK based on the style
of programming (reactive or non-reactive) you wish to adopt in
your application code.

The KAR Java SDK is structured internally into three sub-modules:
1. `kar-runtime-core` - Defines the core abstractions of the
    KAR actor-based programming model in Java and as much as
    possible of the underlying runtime system that implements it.
2. `kar-runtime-liberty` - An implementation of the KAR SDK
    using Open Liberty as the underlying server framework that provides
    a fairly traditional Java programming model based on blocking
    procedure calls and javax.ws.
3. `kar-runtime-quarkus` - An implementation of the KAR SDK
   using Quarkus as the underlying server framework that provides a
   reactive Java programing model based on the Mutiny and Vertx
   frameworks that are used by Quarkus.

To use the Java SDK in an application component, you declare a maven
dependency on one of `kar-runtime-liberty` or `kar-runtime-quarkus` as
described in more detail in the sections below.  You then follow the
framework-specific instructions on using annotations and configuration files
to configure your component (eg. by specifying the Actor types).

The developer-facing APIs for the KAR SDK are split between
`kar-runtime-core` and `kar-runtime-liberty`/`kar-runtime-quarkus`.
The primary API is defined by `com.ibm.research.kar.Kar`
and is specific to the choice of reactive or non-reactive styles.
The primary API is supported by types and annotations
in `com.ibm.research.kar.actor` and its sub-packages which is provided by
`kar-runtime-core`. The package `com.ibm.research.kar.runtime`
defines internal APIs that are not intended for developer use; they
are only made public to enable them to be invoked from framework-specific
runtime code found in `kar-runtime-liberty` and `kar-runtime-quarkus`.

## Building

Builds are driven by maven.  The basic commmand is `mvn install`.

# Using the non-reactive Open Liberty based KAR SDK

Note: the KAR SDK and example code have been tested using MicroProfile
3.3 and the Open Liberty Plugin 3.2 (which pulls v20.0.0.X of
openliberty). You should not use v20.0.0.11 because of a known bug in
the Microprofile Rest Client.

The following code examples show how to use the non-reactive Kar SDK.

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

    JsonValue value = Services.call("MyService", "increment", params);
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

    ActorInstance actor = Actors.ref("ActorType", "ActorID");
    JsonValue value = Actors.call(actor, "remoteMethodName", params);
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

   CompletionStage<JsonValue> cf = Services.callAsync("MyService", "increment", params);

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

    ActorInstance actor = Actors.ref("ActorType", "ActorID");
    CompletionStage<JsonValue> cf = Actors.callAsync(actor, "remoteMethodName", params);

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

## Defining an application component using the Open Liberty based KAR SDK

In addition to writing the framework independent application code
using the KAR SDK, you will need to write some additional bits of
boilerplate to enable Open Liberty to execute your component.

1. You will need to add a stanza to your `pom.xml` to declare a
   dependency on `kar-runtime-liberty` and on some Open Liberty
   dependencies used within KAR. You will also need to include
   the `liberty-maven-plugin` and `maven-war-plugin` to your build plugins.
2. You will need to provide a Java class that extends
   `javax.ws.rs.core.Application`.
3. If your application component contains any KAR Actor types, you
   will need to specify them in your `web.xml` by providing
   values for the `kar-actor-classes` and `kar-actor-types` context params.

For an automatically tested and complete example of doing this, you
should consult [actors-dp-java](../examples/actors-dp-java). In
particular model your configuration on [pom.xml](../examples/actors-dp-java/pom.xml)
and [web.xml](../examples/actors-dp-java/src/main/webapp/WEB-INF/web.xml).

# Using the reactive Quarkus based KAR SDK

The following code examples show how to use the reactive Kar Java SDK.

## Invoke a Service:

```java
import javax.json.Json;
import javax.json.JsonObject;
import javax.json.JsonValue;

import static com.ibm.research.kar.Kar.*;

import io.smallrye.mutiny.Uni;

public static void main(String[] args) {
    JsonObject params = Json.createObjectBuilder()
				.add("number",42)
				.build();

    Uni<JsonValue> uni = Services.call("MyService", "increment", params);
    uni.chain(value -> {
     // Do something with the returned value and produce the next Uni
    }).chain(value2 -> {
     ...
}
```

## Call an Actor Method:

```java
import javax.json.Json;
import javax.json.JsonObject;
import javax.json.JsonValue;

import static com.ibm.research.kar.Kar.*;

import io.smallrye.mutiny.Uni;

public static void main(String[] args) {

    JsonObject params = Json.createObjectBuilder()
				.add("number",42)
				.build();

    ActorInstance actor = Actors.ref("ActorType", "ActorID");
    Uni<JsonValue> uni = Actors.call(actor, "remoteMethodName", params);
    uni.chain(value -> {
      // Do something with the result of the call and produce the next Uni
    }).chain(value2 -> {
      ...
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

import com.ibm.research.kar.actor.annotations.Activate;
import com.ibm.research.kar.actor.annotations.Actor;
import com.ibm.research.kar.actor.annotations.Deactivate;
import com.ibm.research.kar.actor.annotations.Remote;

@Actor
public class MyActor implements ActorInstance {

    @Activate // optional actor constructor
    public Uni<Void> init() {
        // init code
    }

    // Expose this method to the actor runtime.
    // KAR synchronizes requests to the actor by default
    @Remote
    public Uni<Void> updateMyState(JsonObject json) {
        // remote code
    }


    // Expose this method to the actor runtime.
    // KAR synchronizes requests to the actor by default
    @Remote
    public Uni<String> readMyState() {
        // read-only code
    }

    // methods not annotated as @Remote are
    // not callable by actor runtime
    public void cannotBeInvoked() {
    }

    @Deactivate // optional actor de-constructor
    public Uni<void> kill() {
    }

    //.... ActorInstance implementation would be below
    //.....
}
```

## Defining a complete application component using the Quakus-based SDK

In addition to writing the actual application code
using the KAR SDK, you will need to write some additional bits of
configuration to enable Quarkus to execute your component.

1. You will need to add a stanza to your `pom.xml` to declare a
   dependency on `kar-runtime-core`, `kar-runtime-quarkus` and on
   some Quarkus dependencies used within KAR. You will also need to
   add the `quarkus-maven-plugin` and `maven-surefire-plugin` to your build plugins.
2. If your application component contains any KAR Actor types, you
   will need to specify them in your `application.properties` by
   providing values for `kar.actors.type` and `kar.actors.classes`.

For an automatically tested and complete example of doing this, you
should consult [actors-dp-java-reactive](../examples/actors-dp-java-reactive). In
particular model your configuration on [pom.xml](../examples/actors-dp-java-reactive/pom.xml)
and [application.properties](../examples/actors-dp-java-reactive/src/main/resources/application.properties).
