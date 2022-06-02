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

# Overview

This document describes support for using Prometheus and Grafana to
visualize performance metrics for KAR deployment on K3D as well as
for accessing raw KAR sidecar metrics.

Additional configuration is required for each application so that
Prometheus will know which pods to scrape and for application
specific Grafana dashboards.


## Deploying KAR Metrics on K3D

KAR metrics must be deployed after ../scripts/kar-k8s-deploy.sh is run.
**Note** that the "--agent" option should be used with kar-k8s-deploy.sh in order
to embed the prometheus exporter into kafka to expose kafka metrics.

Other Prometheus and Grafana components are deployed with:
```shell
./scripts/metrics/start-kar-metrics.sh
```
This script creates a new K3D node and a new namespace for these components.

Application specific configuration can now be deployed, before or after the
application itself is deployed.
See for example `XXXXXXXXXXXXXXXX` for how this is done for the reefer application.


## Accessing raw KAR sidecar metrics

A KAR sidecar accumulates metrics for all actor and service methods called
in the application container it is managing. Raw metrics are accessed via
the sidecar's runtime_port used by the application to connect to KAR.
By default this is port 3500.

Kubectl port-forward can be used to access a pod's sidecar port by, for example:
```shell
kubectl port-forward -n prometheus pod/{pilosopher-pod-name} 3500:3500
```
Then capture the current snapshot of accumulating metrics into a file with:
```shell
wget localhost:3500/metrics
```
Metrics for the "eat" method are shown with:
```shell
$ grep -e "Philosopher:/eat" metrics 
kar_user_code_invocation_durations_histogram_seconds_bucket{path="Philosopher:/eat",le="0.01"} 11
kar_user_code_invocation_durations_histogram_seconds_bucket{path="Philosopher:/eat",le="0.02"} 21
kar_user_code_invocation_durations_histogram_seconds_bucket{path="Philosopher:/eat",le="0.04"} 65
kar_user_code_invocation_durations_histogram_seconds_bucket{path="Philosopher:/eat",le="0.08"} 143
kar_user_code_invocation_durations_histogram_seconds_bucket{path="Philosopher:/eat",le="0.16"} 304
kar_user_code_invocation_durations_histogram_seconds_bucket{path="Philosopher:/eat",le="0.32"} 624
kar_user_code_invocation_durations_histogram_seconds_bucket{path="Philosopher:/eat",le="0.64"} 1262
kar_user_code_invocation_durations_histogram_seconds_bucket{path="Philosopher:/eat",le="1.28"} 2029
kar_user_code_invocation_durations_histogram_seconds_bucket{path="Philosopher:/eat",le="2.56"} 2029
kar_user_code_invocation_durations_histogram_seconds_bucket{path="Philosopher:/eat",le="5.12"} 2029
kar_user_code_invocation_durations_histogram_seconds_bucket{path="Philosopher:/eat",le="+Inf"} 2029
kar_user_code_invocation_durations_histogram_seconds_sum{path="Philosopher:/eat"} 1035.9611836939996
kar_user_code_invocation_durations_histogram_seconds_count{path="Philosopher:/eat"} 2029
```

Prometheus is configured to scrape the sum and count values to visualize the method's average latency over time.







