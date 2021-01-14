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

A Helm Chart to deploy a dev-mode KAR runtime onto a cluster.

For detailed instructions, see [getting-started.md](../docs/getting-started.md).

### Components deployed

1. Core KAR
   - Sidecar injection machinery (MutatingWebHook)
   - Secrets containing runtime configuration
2. Supporting Components
   - Redis
   - Kafka (and Zookeeper)

KAR can also be configured to use external Kafka and/or Redis instances by
overriding the default settings from `values.yaml`. For example, to use and
external Kafka set `kafka.internal` to `false` and provide all of the values in
the `kafka.externalConfig` structure.
