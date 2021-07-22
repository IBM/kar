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

# KAR: A Runtime for the Hybrid Cloud

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
