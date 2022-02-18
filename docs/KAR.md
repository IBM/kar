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

# Programming Model

KAR makes is possible to construct scalable, fault-tolerant, cloud-native
applications by building networks of loosely coupled application components.

Application components communicate over Kafka and persist state in Redis.
However, they do not interface with Kafka or Redis directly but via the KAR
runtime processes. A KAR runtime process runs alongside each application
component. It offers REST APIs that permit one component to invoke another
(Remote Procedure Call), to produce or consume events, and to persist state.
While these REST APIs may be used directly, KAR provides language-specific SDKs
that offer more idiomatic APIs and facilitate the implementation of actors. A
JavaScript and a Java SDK are available now, with more to come in the near
future. In the future, we may also consider offering alternatives to Kafka and
Redis.

In this document, we define and illustrate KAR's key concept using a collection
of [examples](../examples). We assume a working KAR deployment. See [KAR
Deployments](kar-deployments.md) for options. The full REST API documentation is
available [here](https://ibm.github.io/kar/api/redoc/).

## Applications

A KAR _application_ is identified by its name. KAR supports running multiple
independent applications concurrently. Distinct applications may communicate by
means of _events_, i.e., publishing events and/or subscribing to events.

A KAR application consists of a dynamic set of _components_ akin to running
instances of microservices, _event sources_ and _sinks_, and a _persistent
store_.

KAR uses name mangling to allow for a single Redis instance and single Kafka
instance to support multiple applications without unintended interference.

Most KAR CLI commands requires specifying the name of the application to target,
for example:
```
kar purge -app demo
```
This command purges the state of the `demo` application from Kafka and Redis.

## Components

An application _component_ is a unit of compute and state. A component belongs
to a single application. A component is _joined_ to a specific application at
launch time by providing the name of the application (required). The component
belongs to the same application until it terminates. The set of components in an
application vary over time as new components join the application or existing
components terminate.

Components can be stateless or stateful. Stateful components intended to be
scalable or fault-tolerant should either manage their state on their own or
leverage KAR's actors and persistent store.

A component can be pretty much anything. In this tutorial, we will encounter
examples of components built using Node.js, OpenLiberty, and curl. Individual
components may be deployed as simple OS processes or as containers running on
platforms such as Docker, Kubernetes, OpenShift, and IBM Code Engine.

## Ports

A KAR runtime process runs alongside each component process. We distinguish two
kinds of components:
- a _client_ component issues HTTP requests to the corresponding KAR runtime
  process.
- a _server_ component can not only issue HTTP requests to the KAR runtime
  process, but also handle HTTP requests issued by the KAR runtime process.

The KAR runtime process is listening on the _runtime port_. The server process
is listening the _app(lication) port_.

By default the KAR runtime port is autoselected and the KAR application port is
set to 8080. When multiple server components are running on a single host, the
default KAR application port must be overridden to avoid conflicts.

## Launchers

Using the KAR CLI, a KAR component is typically launched as follows:
```
kar run -app demo -- node demo-component.js
```
This command launches the KAR component process as well as the matching KAR
runtime process. This component is joined to the `demo` application.

Using `kubectl` a KAR component may be launched with a YAML specification such
as:
```
apiVersion: v1
kind: Pod
metadata:
  name: demo-component
  annotations:
    kar.ibm.com/app: demo
spec:
  containers:
  - name: demo-component
    image: demo-component-image
```
This YAML launches both the containerized KAR component as well as the KAR
runtime process as a sidecar container. The application name is specified as an
annotation.

In both cases, the component code may obtain the runtime and application port
numbers by reading the environment variables `KAR_RUNTIME_PORT` and
`KAR_APP_PORT`. The default port numbers may be overridden using the
`-runtime_port` and `-app_port` flags of the `kar run` command or by adding the
`kar.ibm.com/runtimePort` and `kar.ibm.com/appPort` annotations to the YAML
specification.

## Services

An application component may offer a single _service_ identified by its name,
specified at launch time. A component is not required to offer service. The
component offers the same service (if any) until it terminates. Multiple
components of the same application may offer the same service, akin to multiple
replicas of a microservice.

A component that offers a service must implement a REST server. For instance
[server.js](../examples/service-hello-js/server.js) in example
[service-hello-js](../examples/service-hello-js) implements a KAR service in
JavaScript using [Express](https://expressjs.com) for Node.js. Alternatively,
the [service-hello-java](../examples/service-hello-java) code implements the
same example in Java using OpenLiberty.

Launching a KAR service requires specifying the application name and service
name, for instance using the KAR CLI:
```
kar run -app hello-js -service greeter -- node server.js
```
When deploying to Kubernetes or OpenShift, these names are provided by means of
annotations, for example:
```
apiVersion: v1
kind: Pod
metadata:
  name: hello-server
  annotations:
    kar.ibm.com/app: hello-js
    kar.ibm.com/service: greeter
spec:
  containers:
  - name: server
    image: localhost:5000/kar/kar-examples-js-service-hello
```
Ideally the server should be listening on the port specified by the
`KAR_APP_PORT` environment variable, for instance
[server.js](../examples/service-hello-js/server.js) includes code:
```
app.listen(process.env.KAR_APP_PORT)
```
Otherwise the application port must be specified when launching the runtime
process for the component.

## Requests

An application component can make a _request_ to a service of the application. A
request has the usual elements of an HTTP request: a method, a route, a payload,
and headers. KAR delivers the request to any component offering the service. If
no such component is available KAR persists the request until it can be
delivered (up to a configurable timeout).

Requests may be _synchronous_ or _asynchronous_. Requests are synchronous by
default. A synchronous request returns the _response_ to the HTTP request. An
asynchronous request returns as soon as KAR accepts the request. If desired, a
request id can be returned from an asynchronous request to permit querying KAR
for the response later.

## Requests: REST API

The KAR runtime process exposes a REST API to support service requests.

For instance, assuming the `greeter` service of application `hello-js` is
running, we can join a component to this application to make a request using
`curl`.
```
kar run -app hello-js -- sh -c 'curl -s -X POST -H "Content-Type: text/plain" http://localhost:$KAR_RUNTIME_PORT/kar/v1/service/greeter/call/helloText -d "Gandalf the Grey"'
```
```
2020/12/02 10:25:37.284552 [STDOUT] Hello Gandalf the Grey!
```
This component makes a single synchronous request to the `greeter` service,
outputs the response, and terminates. The URL for the request follows the schema
specified in the [KAR REST API
documentation](https://ibm.github.io/kar/api/redoc/). It includes
the target service name `greeter` and route `helloText`. The request also
specifies the method `POST`, the payload `"Gandalf the Grey"`, and headers.

The `KAR_RUNTIME_PORT` environment variable is automatically set by the KAR
launcher to the port of the KAR runtime process. We wrap the `curl` command with
a shell invocation `sh -c` to ensure proper expansion of this variable.

An asynchronous request is obtained by adding a `Pragma` header to the request.
If the `Pragma: Async` header is specified (case insensitive), the request
simply returns `Accepted`. If the `Pragma: Promise` header is specified (case
insensitive), the request returns a request id. See [KAR API
documentation](https://ibm.github.io/kar/api/redoc/) for details.

## Requests: CLI

Because making requests from a terminal is very useful when developing KAR
services, the KAR CLI offers a convenient shorthand for synchronous requests:
```
kar rest -app hello-js -content_type text/plain post greeter helloText 'Gandalf the Grey'
```
```
Hello Gandalf the Grey!
```

## Requests: Javascript SDK

The JavaScript SDK offers convenience methods to make synchronous and
asynchronous requests with method `POST` and headers `Content-Type:
application/json`:
```
const { call, tell, asyncCall } = require('kar')

async function main () {
  // synchronous request to a service
  console.log(await call('greeter', 'helloJson', { name: 'Alice' }))

  // asynchronous request
  console.log(await tell('greeter', 'helloJson', { name: 'Bob' }))

  // asynchronous request returning a handle
  const resolve = await asyncCall('greeter', 'helloJson', { name: 'Charlie' })

  // waiting for the response to the asynchronous request
  console.log(await resolve())
}

main()
```
```
kar run -app hello-js -- node client.js
2020/12/10 08:44:14.554886 [STDOUT] { greetings: 'Hello Alice!' }
2020/12/10 08:44:14.559342 [STDOUT] OK
2020/12/10 08:44:14.567674 [STDOUT] { greetings: 'Hello Charlie!' }
```

## Persistent Store

An application includes a _persistent store_. Application components can
_create_, _read_, _update_, and _delete_ content from the application store. KAR
also leverages this store to implicitly persist other things such as the state
of in-progress service requests.

An application starts when a first component is joined to the application. Its
store is initially empty. While the number of components in an application may
go down to zero, KAR will persists its store content and make it available to
components later joining the application.

Since KAR may persist state indefinitely, components should make sure to delete
unnecessary content from the persistent store.

The KAR CLI makes it possible to purge the application state:
```
kar purge -app demo
```
The `purge` command eliminates the application state in Redis and Kafka. It
should only be invoked when no application component is running.

Alternatively, the `drain` command preserves the state explicitly created by the
application but drops all pending requests. Intuitively, it flushes Kafka queues
for the target application (and the corresponding runtime state in Redis) but it
preserves the application state Redis.
```
kar drain -app demo
```

## Actors

KAR includes a virtual actor model that provides system-managed stateful
entities. An application component can support one or more _actor types_ and
host many _actor instances_ of the supported types. The entire actor lifecycle
of these actor instances is managed by the KAR runtime.

An actor instance (or actor in short) is a logically independent unit of compute
and state, typically much smaller than an application component. An actor
instance offers _methods_ that can query and/or update the state of the actor
instance and invoke other actors. Actors offer a single-threaded execution model
where two method invocations on an actor instance may not make progress
concurrently.

Every actor instance is an instance of an actor type. Actor instances of the
same type are expected to offer the same methods and logically represent
entities of the same kind but with different state.

An application component never constructs, destructs, or invokes methods on
actor instances directly. An application component can only invoke an actor
method by means of an _actor reference_. An actor reference consists of an actor
type and an alphanumerical _actor ID_. Any application component can construct
an actor reference by combining an actor type and an arbitrary ID. Actor
references of different types may share the same ID but enjoy no special
relationship.

Application components may host many actor instances of multiple actor types.
Application components must specify at launch time what actor types they
support. A application component supporting an actor type T, must implement a
REST server capable of handling the following requests:
* construct an actor instance with type T and a given ID,
* invoke a method on the actor instance with type T and a given ID,
* destruct an actor instance with type T and a given ID.

KAR includes SDKs to facilitate the implementation of such REST servers. For
instance, the JavaScript SDK for KAR makes it possible to define an actor type
by means of a class declaration. An actor instance is simply an instance of that
class (a object) and its methods are the methods of the actor. Moreover, the
fields of this object hold the in-memory state of the actor instance. Multiple
actor types may be defined by means of multiple classes. The SDK handles the
mapping from actor types and IDs to object references and implements the
required routes of a REST server.

The [philosophers.js](../examples/actors-dp-js/philosophers.js) file in the
[actors-dp-js](../examples/actors-dp-js) example demonstrates how to define and
serve four actor types using KAR's Javascript SDK:
```
const { sys } = require('kar')

class Fork { ... }
class Philosopher { ... }
class Table { ... }
class Cafe { ... }

const app = express()
app.use(sys.actorRuntime({ Fork, Philosopher, Table, Cafe }))
app.listen(process.env.KAR_APP_PORT)
```
The `sys.actorRuntime` helper method turns the four actor classes into the
required Express middleware (HTTP handlers).

In general, `sys.actorRuntime({ actorType: className, ... })` defines the actor
type `actorType` by means of the class `className` making it possible for the
two names to differ. In practice, we recommend against it.

For now, the names of the provided actor types must be repeated when launching
the application component using flag `-actors`:
```
kar run -app dp -actors Cafe,Table,Fork,Philosopher -- node philosophers.js
```
An application component may offer a service and at the same time hosts actors
by implementing a REST server capable of doing both.

KAR SDKs also embed the actor instance type and id into the actor instance.
Using the Javascript SDK, if `myActorInstance` is an actor instance, then
`myActorInstance.kar.type` and `myActorInstance.kar.id` respectively provide the
actor type and id for this instance.

## Actors: State

KAR provides a range of APIs to save and retrieve persistent data associated
with an actor instance from the persistent store. This data for each actor
instance is organized as a key-value map. The API supports different levels of
granularity with whole map, single key, and sub-map operations.

For instance, the JavaScript SDK offers the following methods: `getAll`,
`setMultiple`, `removeAll`, `get`, `set`, `remove`, `contains`, `setWithSubkey`,
`setMultipleInSubMap`, `subMapGetKeys`, `subMapGet`, `subMapGetSize`,
`subMapClear`.

## Actors: Lifecycle

When a method is invoked on an actor reference, KAR first checks whether a
matching instance exists for this reference. If there is no such instance, KAR
selects an application component that supports the actor type specified in the
method invocation and asks this component to construct an actor instance with
the given type and ID. It then invokes the method on this actor instance with
the specified arguments. Hence, by design, KAR will never invoke a method on an
actor instance that has not been constructed first.

By convention, KAR SDKs invoke the `activate` method on an actor instance
immediately after construction if this method exists on the actor instance. The
SDKs invoke the `deactivate` method of the actor instance if it exists
immediately before reclaiming the instance.

For instance, the `activate` method of the `Fork` actor class in
[philosophers.js](../examples/actors-dp-js/philosophers.js) attempts to restore
the state of the `Fork` instance from the persistent store:
```
async activate () {
  this.inUseBy = await actor.state.get(this, 'inUseBy') || 'nobody'
}
```
Like service invocations, method invocations on actors can be synchronous and
asynchronous. A synchronous method invocation returns the result of the
invocation of the method on the actor instance to the caller. An asynchronous
method invocation returns as soon as KAR accepts the invocation. If desired, a
request id can be returned from an asynchronous method invocation to permit
querying KAR for the response later. See [KAR API
documentation](https://ibm.github.io/kar/api/redoc/) for details.

If no application component is available to instantiate the specified actor
type, KAR persists the invocation request until it can be delivered (up to a
configurable timeout).

The [philosophers.js](../examples/actors-dp-js/philosophers.js) file contains
many examples of actor invocations using the Javascript SDK, for instance:
```
// call: synchronous invocation
await actor.call(actor.proxy('Table', this.table), 'doneEating', this.kar.id)

/// tell: asynchronous invocation
await actor.tell(this, 'serve', servings, step)
```
These invocations take as parameters:
- a reference to the target actor,
- the actor method,
- zero, one, or multiple arguments.

The actor reference is either constructed from the actor type and id using
`actor.proxy` or the shorthand `this` when an actor instance is calling a method
on itself.

Subsequent method invocations on the same actor reference will normally be
handled by the same actor instance in the same application component. But, if an
actor instance is not used for a period of time, KAR will request the
application component to destruct the actor instance. If a method is later
invoked on the same actor reference, KAR will first construct a new actor
instance for this invocation, often but not always hosted by the same
application component.

Because KAR makes explicit requests to construct and destruct actor instances to
application components, these components can save the actor instance state to
the application persistent store before destruction and restore the actor
instance state upon reconstruction in order to give the illusion of a persistent
actor instance.

KAR may migrate an actor instance from one application component to another when
no method is running on the instance by first destructing the existing instance
then creating a new instance.

## Actors: Sessions

A method invocation on an actor reference may optionally include an
alphanumerical _session ID_. If no session ID is specified, KAR synthesizes a
random unique ID.

KAR ensures that concurrent invocations of methods on actor instance with
distinct session IDs are executed serially by queuing invocations if necessary.
On the other hand, method invocations bearing the same session ID may execute
concurrently.

The primary intent of session IDs is to support actor re-entrancy. Re-entrancy
makes it possible for an actor instance to make a synchronous invocation of a
method on itself (possibly indirectly) without deadlocking. KAR actor SDKs
implicitly thread session IDs through method invocations to enable re-entrancy.
Concretely, when an actor method invokes another actor method synchronously (for
the same actor reference or a different one), it reuses the session ID it has
been invoked with.

While re-entrancy permits method invocations to overlap, it still ensures that
no two invocations make progress concurrently, since only nested synchronous
invocations share the same session ID.

### Actors: Reminders

A _reminder_ is a time-triggered asynchronous invocation of an actor
method.  The system supports both _one-shot_ reminders that are scheduled to
run once after their target time arrives and _periodic_ reminders that
are automatically rescheduled to run again with a specified delay
after they are delivered. Both one-short and periodic reminders are
implicitly persisted by the KAR runtime. Reminders will continue to
fire even if an actor instance is lost or destructed, reconstructing
the actor instance at firing time if necessary.

The [philosophers.js](../examples/actors-dp-js/philosophers.js) file contains
several examples of actor reminders using the JavaScript SDK, for instance:
```
await actor.reminders.schedule(this, 'getFirstFork', { id: 'step', targetTime: this.nextStepTime() }, 1, step)
```
In this snippet, the currently running actor instance schedules an
invocation of `getFirstFork(1, step)` on itself to occur at a
targetTime of `this.nextStepTime()`. This is a one-shot reminder.

The [ykt.js/(../examples/actors-ykt/ykt.js) contains an example of a
periodic reminder:
```
await actor.reminders.schedule(actor.proxy('Site', site), 'siteReport',
    { id: 'aisle14', targetTime: new Date(Date.now() + 1000) }, '5s')
```
Here an invocation of `siteReport()` is scheduled to be delivered to
the `site` instance of the `Site` actor every 5 seconds, with the first
invocation happening 1 second in the future.

## Events

KAR provides applications with a publish/subscribe sub-system that can be bound
to a variety of concrete event sources and sinks using Camel.

Application components can publish events to a _topic_ identified by its name.
Actor instances can subscribe to a _topic_ by specifying a method to invoke on
each event delivered to this topic.

Topic names are global. In other words, distinct applications can communicate by
emitting and receiving events on the same topics.

Subscriptions are identified by a _subscription ID_. KAR provides APIs to not
only create subscriptions, but also query, update, and delete existing
subscriptions.

Subscriptions are implicitly persisted by the KAR runtime. Subscriptions will
continue to deliver events even if an actor instance is lost or destructed,
reconstructing the actor instance on event arrival if necessary.
