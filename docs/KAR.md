# KAR Programming Model

## Applications

A KAR _application_ is identified by its name. KAR supports running multiple
independent applications concurrently. Distinct applications may communicate by
means of _events_, i.e., publishing events and/or subscribing to events.

A KAR application consists of a dynamic set of _components_ akin to running
instances of microservices and a _persistent store_. A _component_ is a unit of
compute and state. A component belongs to a single application. A component is
_joined_ to a specific application at launch time by providing the name of the
application (required). The component belongs to the same application until it
terminates. The set of components in an application vary over time as new
components join the application or existing components terminate.

## Services

An application component may offer a single _service_ identified by its name,
specified at launch time (optional). The component offers the same service until
it terminates. Multiple components of the same application may offer the same
service, akin to multiple replicas of a microservice. A component is not
required to offer service. A component that offers a service must implement a
REST server.

An application component can make a _request_ to a service of the application. A
request consists of the name of the target service and an HTTP request. KAR
delivers the request to any component offering the service. If no such component
is available KAR persists the request until it can be delivered. Requests may be
_synchronous_ or _asynchronous_. A synchronous request returns the _response_ to
the HTTP request. An asynchronous request returns as soon as KAR accepts the
request with a simple _acknowledgment_.

## Stores

An application includes a _persistent store_. Application components can
_create_, _read_, _update_, and _delete_ content from the application store. KAR
also leverages this store to implicitly persist other things such as the state
of in-progress service requests.

An application starts when a first component is joined to the application. Its
store is initially empty. While the number of components in an application may
go down to zero, KAR will persists its store content and make it available to
components later joining the application.

## Actors

KAR provides support for _actors_. An _actor instance_ (or actor in short) is a
logically independent unit of compute and state, typically much smaller than an
application component. An actor instance offers _methods_ that can query and/or
update the state of the actor instance and invoke other actors. Actors offer a
single-threaded execution model where two method invocations on an actor
instance may not make progress concurrently.

Every actor instance is an instance of an _actor type_. Actor instances of the
same type are expected to offer the same methods and logically represent
entities of the same kind but with different state.

An application component never constructs, destructs, or invokes methods on
actor instances directly. An application component can only invoke an actor
method by means of an _actor reference_. An actor reference consists of an actor
type and an alphanumerical _actor ID_. Any application component can construct
an actor reference by combining an actor type and an arbitrary ID. Actor
references of different types may share the same ID but enjoy no special
relationship.

Application components must specify at launch time what actor types they
support. A application component supporting an actor type T, must implement a
REST server capable of handling the following requests:
* construction of an actor instance with type T and a given ID,
* invocation of a method on the actor instance with type T and a given ID,
* destruction of an actor instance with type T and a given ID.

KAR includes actor SDKs to facilitate the implementation of such REST servers.
For instance, the JavaScript SDK for KAR makes it possible to define an actor
type by means of a class declaration. An actor instance is simply an instance of
that class and its methods are the methods of the actor. The SDK handles the
mapping from actor IDs to object references.

Application components may support multiple actor types.

### Actor Lifecyle

When a method is invoked on an actor reference, KAR first checks whether an
matching instance exists for this reference. If there is no such instance, KAR
selects an application component that supports the actor type specified in the
method invocation and asks this component to construct an actor instance with
the given type and ID. It then invokes the method on this actor instance with
the specified arguments. Hence, by design, KAR will never invoke a method on an
actor instance that has not been constructed first.

Method invocations can be synchronous and asynchronous. A synchronous method
invocation returns the result of the invocation of the method on the actor
instance to the caller. An asynchronous method invocation returns as soon as KAR
accepts the invocation with a simple acknowledgment.

If no application component is available to instantiate the specified actor
type, KAR persists the invocation request until it can be delivered.

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

## Sessions

A method invocation on an actor reference may optionally include an
alphanumerical _session ID_. If no session ID is specified, KAR synthesizes a
random unique ID.

KAR ensures that concurrent invocations of methods on actor instance from
different sessions are executed serially by queuing invocations if necessary. On
the other hand, method invocations bearing the same session ID may execute
concurrently.

While session IDs may be managed by callers explicitly and serve arbitrary
purposes, the primary intent of session IDs is to support actor re-entrancy.
Re-entrancy makes it possible for an actor instance to make a synchronous
invocation of method on itself (possibly indirectly) without deadlocking. KAR
actor SDKs implicitly thread session IDs through method invocations to enable
re-entrancy. While re-entrancy permits method invocations to overlap, it still
ensures that no two invocations make progress concurrently.

## Reminders

TODO

## Events

TODO

## Fault-Tolerance

TODO
