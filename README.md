<!--
# Copyright IBM Corporation 2020,2023
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

# KAR: A Runtime for the Hybrid Cloud

[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0)
[![Continuous Integration](https://github.com/IBM/kar/actions/workflows/ci.yaml/badge.svg)](https://github.com/IBM/kar/actions/workflows/ci.yaml)

The KAR runtime makes it easy to _develop_ and _run_ stateful, scalable,
resilient applications for the _hybrid cloud_ that combine microservices and
managed services.

KAR is:
- open source and vendor neutral.
- cloud-native: KAR is built for Kubernetes and OpenShift.
- polyglot: KAR supports any programming language and developer framework by
  means of REST APIs. Idiomatic SDKs for specific languages may be developed
  with minor effort.
- simple yet expressive: KAR interfaces stateless and stateful microservices
  using requests and events.
- scalable: KAR is designed from the ground up to handle dynamic scaling of
  replicated stateless and stateful microservices.
- resilient: KAR combines persistent message queues with persistent data stores
  to offer strong fault-tolerance guarantees.
- extensible: KAR applications can produce or consume events and data streams
  using hundreds of [Apache Camel](https://camel.apache.org) sources and sinks.

# Scalable and Fault-Tolerant State

KAR puts a great deal of emphasis on helping developers manage application
state. Stateless microservices are easy to scale and easy to restart or replace
on failure. Stateful microservices are not. Moreover the state of an application
not only includes the state of its microservice components, but also the state
of in-flight requests or events, external state in databases or on disk, etc.
Keeping track of this state, avoiding performance bottlenecks, and protecting it
from failures is typically very hard.

## Actors

KAR make it easy to structure the state of microservices as a collection of
_actor_ instances. The [actor model](https://en.wikipedia.org/wiki/Actor_model)
is a popular and well-understood approach to programming concurrent and
distributed systems. Each actor instance is responsible for its own state. The
state of an actor instance can be saved or restored safely (because actor
instances are single-threaded) and independently from other actor instances
(since there is no shared state).

KAR offers simple APIs for actors to incrementally save their state to Redis.
These APIs can be triggered periodically, or when idle with little effort. KAR
can automatically restore the state of a failed actor instance. Timers or event
subscriptions associated with an actor instance are also restored.

Actor instances can migrate from one microservice replica to another due to
failures or for load balancing purposes. KAR understands that actors are
relocatable. KAR's API for invoking actors transparently routes, and if necessary
reroutes, requests to the proper destination.

For instance, in the simulation engine example  described below, the simulation state is
partitioned across multiple replicas of the simulator microservice using actors.
A developer can reason about and program these actors and their interactions
without having to worry about exhausting the resources of a single process or
mapping actor instances to processes. In that sense, KAR supports a "serverless"
experience.

## Retry Orchestration

KAR automatically retries failed (i.e., interrupted) actor method invocations.
Retries are necessary but dangerous. Many other systems
proactively retry a task when its success is in
doubt, for
instance if it has not completed by a deadline. As a result, multiple executions
of a task may happen concurrently. Worse, two tasks in a sequence may end up
running concurrently as a spurious retry of the first one overlaps with the
second. The tasks therefore have to be carefully engineered to be resilient not
only to sequential retries, but also concurrent retries, and possible
reordering. By contrast, KAR is designed to better orchestrate retries---retries
are more constrained---so as to unburden developers from complex non-local
reasoning.

To start with, KAR guarantees that:
- a failed invocation is retried until success
- a successfully completed invocation is never retried
- a strict happens before relationship is preserved across failures within each distributed chain of invocations and retries.

In other words, KAR will try as many times as necessary, making one attempt
after the other, but not once more than necessary.
KAR goes beyond individual invocations to offer guarantees about nested
invocations and chains of invocations.
- KAR guarantees that a retry of a failed caller will not begin until all of the
non-failed callees of the previous execution have completed.
- KAR introduces a tail call
mechanism that makes it possible to transactionally transfer control from one
actor method to another (of the same or a different actor) so that in a chain of
invocations, only the last invocation in the chain will be retried even if both
the caller and callee actors have failed. Developers still have to worry about
retries, typically by making individual actor methods idempotent, but, using tail
calls, complex code can be broken into smaller pieces that are easier to make
idempotent.

KAR strives to achieve such guarantees in a dynamic, distributed system with
minimal overheads.

For a detailed technical description, see [Reliable Actors with Retry Orchestration](https://arxiv.org/abs/2111.11562).

# KAR Application Mesh

KAR is deployed as a lightweight process, a container, or a Kubernetes sidecar
that runs alongside each microservice:
- The KAR process exposes a REST API to the microservice. Using this API, the
  microservice can make synchronous and asynchronous requests to other
  microservices, produce or consume events, or manage its persistent state.
- This REST API is served over HTTP/1.1 for maximal compatibility as well as
  HTTP/2 for high performance and scalability.

Together the KAR processes form a mesh:
- This mesh can run entirely on a developer's laptop, or entirely within a single Kubernetes cluster,
  or spanning multiple clusters, servers, VMs, edge devices, etc.
- This mesh leverages Kafka to decouple the microservices from one another and
  guarantee reliable request/response and publish/subscribe interactions.
- This mesh has no leader, no single point of failure, and no external dependency other than
  a Kafka and Redis instances.

![KAR](docs/images/mesh.png)

Using the KAR mesh, a typical application interfaces a collection of
microservices, event sources, event sinks, and interactive client/CLI processes.
Consider for instance the architecture of the simulation engine described in
[actor-ykt](examples/actors-ykt/README.md). This application combines:
- a replicated simulator microservice that can be scaled to accommodate many
  simulated agents.
- a singleton reporter microservice that produces reports on a schedule or on
  demand.
- a controller that runs only when a human operator is controlling the
  simulator.
- a notifier that sends reports to a Slack channel.

The simulator, reporter, and controller are Node.js components implemented in
JavaScript. The notifier component leverages the Camel runtime and is configured
by means of a few lines of YAML.

A developer may choose to deploy the simulator to Kubernetes/OpenShift but run
the controller on his laptop. The KAR CLI or operator automatically injects and
configures the KAR runtime that runs alongside each component.

# Quick Links

+ See [Getting Started](docs/getting-started.md) for hands-on instructions for
  trying KAR.
+ See [KAR Deployment Options](docs/kar-deployments.md) for detailed instructions
  for deploying KAR-based applications on a wide range of platforms.
+ Check out our [examples](examples/README.md).
+ Read about the KAR [Programming Model](docs/KAR.md).
+ Read a technical description about KAR's approach to fault tolerance: [Reliable Actors with Retry Orchestration](https://arxiv.org/abs/2111.11562).
+ Check out some larger [applications](https://github.com/IBM/kar-apps) that use KAR.
+ Browse the Swagger specification of the [KAR API](https://ibm.github.io/kar/api/redoc/).
+ See [Notes for KAR Developers](docs/kar-dev-hints.md) for detailed
  instructions on how to build KAR for source.

# License

KAR is an open-source project with an [Apache 2.0 license](LICENSE.txt).
