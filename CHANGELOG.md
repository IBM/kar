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

# KAR 1.3.8 - 2023-05-15
+ Update k3d ingress setup (#380)
+ Remove placeholder value for redis password (#379)

# KAR 1.3.7 - 2023-05-05
+ Revert #351 - Container Shipping application assumes co-mingled primary/submap state in getAll

# KAR 1.3.6 - 2023-05-03
+ General refresh of dependencies
  + Update Kafka version to 3.3 and ZooKeeper to 3.8
  + Update JavaScript SDK, examples, and Dockerfiles to use Node.js 18
  + Update Java SDKs to OpenLiberty 23.0.0.3 and Quarkus 2.16.Final
  + Switch from adoptopenjdk to eclipse-temurin as base Java Dockerfiles
  + Update core docker image to alpine 3.17
  + Update core to go 1.19
  + Update scripts to use kind 0.18 with Kubernetes 1.24
+ Bug fixes
  + TailCalls should have an infinite request deadline (#350)
  + Segregate primary and submap state in getall (#351)
+ Changed project CI from TravisCI to GitHub Actions

# KAR 1.3.5 - 2022-08-05
+ New distributed debugger for KAR (#338, #340)

# KAR 1.3.4 - 2022-07-05
+ Upgrade kind from 0.12.0 to 0.14.0 (#333)
+ Reenable cancellation as an option (#331)
+ Improve documentation for logging and metrics
+ Add documentation for Python SDK. (#325)
+ Tolerate transient failures to connect to Kafka and Redis during startup
+ Add metrics for redis request time and active/canceled reminders (#326)
+ Customize Python version. (#324)
+ Fix activate method call in Python SDK (#323)

# KAR 1.3.3 - 2022-05-05
+ Encode service result when invoked from actor tail call (#316, #317)
+ Bug fix in actor scheduling for recovery from a blocked self-call (#318)
+ Improve debug logging in message processing layer of rpclib (#319)

# KAR 1.3.2 - 2022-04-25
+ Implement basic tail call from Actor to Service (#313)
+ Bug fix in actor locking -- overly forgiving reentrancy bypass (#309)

# KAR 1.3.1 - 2022-04-07
+ Enhancements to the Python SDK
  + Support for reentrancy (#288) (#294)
  + Support for async actor calls (#290) (#293)
  + Support for reminders (#298)
  + Support for tail calls (#296)
+ Enable simple NRU-based actor placement cache (#285)
+ Improvements to KAR deployment scripts
  + Add support for logging stack (#301)
  + Add support for metrics stack (#286) (#292)

# KAR 1.3.0 - 2022-03-22
+ KAR 1.3.0 introduces several major enhancements to the programming model
  + Failure recovery ensures that a retry of a failed actor invocation
    will not be executed until after all synchronous calls made by the
    failed version of the task have completed.
  + Tail calls to the same actor instance retain the actor lock by default.
  + Each actor's queue of incoming messages is now always processed in order.
+ There is a new Python SDK.
+ The legacy transport layer that predated pkg/rpc was removed.

# KAR 1.2.3 - 2022-02-28
+ JavaSDK: fix @Produces annotation on Service.tell* routes in OpenLiberty sidecar (#254)

# KAR 1.2.2 - 2022-02-18
+ Updated npm and maven packages to resolve CVEs (various)
+ Update Kafka version to 2.8.1 (#240)
+ Bump Quarkus from 2.2.3.Final to 2.4.2.Final (#232)
+ Improvements in Java SDKs for getReminders (#238, #245, #246, #247)
+ Improve log message for dropped tells (#249)
+ Fix for default replication factor for Event Streams (#237)
+ Add unit testing infrastructure and RPC library testing (#222)

# KAR 1.2.1 - 2021-11-23
+ Use rpclib by default (#218)
+ Add ability to control topic-level message retention (#216, #217)

# KAR 1.2.0 - 2021-11-09
+ Add tail call support to rpclib, SDKs, and use in actors-dp-* examples
+ Document deploying on k3d (supplanting k3s)
+ Improve microbenchmarks
+ Implement a cache of actor placement info for rpclib

# KAR 1.1.0 - 2021-10-08
+ Implement an alternative Kafka-only RPC layer
    + Port KAR runtime to new rpclib APIs
        + Port KAR to new RPC library abstractions (#170)
        + Promote internal/store to pkg/store (#171)
        + Add alternative store.CAS implementation that returns value instead of Boolean (#173)
    + New Kafka-only RPC library
        + Initial import of rpclib (#174)
        + Subsequent fixes (#177), (#178), (#179), (#180), (#184), (#185), (#186)
+ Adding a transactional framework in KAR (#167)
+ Upgrade from Open Liberty 20.0.0.9 to 21.0.0.7 (#181)
+ Upgrade from Quarkus 1.1.13 to 2.2.3 (#176)
+ Upgrade from kafka 2.7.0 to 2.7.1 (#182)
+ Fix bug introduced in 1.0.9 in redis retry logic (#172)

# KAR 1.0.9 - 2021-09-10
+ Implement a Reactive Java SDK using Quarkus
+ Add a retry loop around failed redis connection attempts (#163)
+ Also allow controlling sidecar ports via envvar (#153)
+ Move from alpine 3.11 to 3.14 for sidecar/webhook images (#151)
+ Fixup kafka-bench and re-enable building it. (#150)

# KAR 1.0.8 - 2021-08-13
+ Upgrade to zookeeper 3.6 and kafka 2.7 (#146)
+ Upgrade to use Redis 6 (#145)
+ Implement cancellation of actor calls from dead sidecars (#144)

# KAR 1.0.7 - 2021-08-09
+ Simplify Java SDK initialization (#139)

# KAR 1.0.6 - 2021-07-30
+ Restructure of Java SDK internals and new maven artifact names
+ Add Prometheus metrics endpoint to sidecar

# KAR 1.0.5 - 2021-07-22
+ Use factory to create JsonObjectBuilder and JsonArrayBuilder instances (#126)
+ Update to kind 0.11.1 (#125)
+ Rationalize concurrency controls for Java actor runtime (#123)

# KAR 1.0.4 - 2021-06-08
+ Support for general multi-element Actor state updates (#117)
+ Optimize submap operations by using HSCAN (#115, #114, #112)
+ Default to Kubernetes 1.20 on kind (#110)
+ Ignore docker network connect error (#109)
+ Always add docker registry to kind network (#107)
+ Allow millisecond granularity periods for reminders (#105)

# KAR 1.0.3 - 2021-04-14
+ Support for persistent volumes and zookeeper/kafka replication (#95)
+ Improve naming and documentation of timeout kar arguments (#82)
+ Java SDK configuration cleanups (#81)
+ Implement infinite service/actor timeout in sidecar (#79)
+ Truncate large backtraces to avoid exceeding Kafka message size (#78)

# KAR 1.0.2 - 2021-03-30
+ support for deploying on OpenShift 4.x (#73)
+ upgrade to zookeeper 3.5 and kafka 2.6

# KAR 1.0.1 - 2021-02-22
+ Add testcases for Java timeout scenario (#61, #62)
+ Java SDK: infinite default timeout (#59)

# KAR 1.0.0 - 2021-02-12
First stable release
