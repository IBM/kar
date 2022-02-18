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

### Example Structure

Each child directory contains an example KAR application.

Each example contains a `deploy` directory that contains
artifacts for deploying the application. Multiple deployment modes may
be supported via a combination of scripts, yaml files, and Helm
charts.  See the README.md in each directory for instructions.

### Examples in a Nutshell

+ A small Greeting service sample shows how to to extend standard
  [Java JAX-RS](./service-hello-java) and
  [NodeJS Express](./service-hello-js) clients and servers to work
  with KAR.

+ We  use the classic Dining Philosophers problem to introduce KAR's
  actor programming model by instantiating the same fault-tolerant
  implementation of Dijkstra's solution to the problem in both
  [Java](./actors-dp-java) and [JavaScript](./actors-dp-js).

+ The [Yorktown Simulation](./actors-ykt) demonstrates using KAR's agent
  model for virtual stateful services to implement a scalable simulation.

+ [Unit Tests](./unit-tests) contains unit tests and scripts to
  execute them.
