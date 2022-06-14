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

This document describes support for using a local logging stack to
capture KAR application logs on K3D and access them with Kibana.

The logstack includes fluentd, elasticsearch and kibana components
which are deployed on a separate K3D node. Logs are collected from
KAR application components deployed on K3D nodes that are labeled with
`kar-type: worker`

## Deploying KAR logstack on K3D

The logstack components are deployed with:
```shell
../scripts/logstack/start-k3d-logstack.sh
```
This script creates a new K3D node if necessary, creates a new namespace,
and then deploys the components. It can be started before or after KAR
application components are deployed.


## Accessing logstack data

Kibana is accessible from port 5601 in the kibana service.
The following steps will give an initial view into the data collected:

   + Expose access to it from outside the K3D cluster with:
```shell
kubectl port-forward svc/kibana  5601:5601 --namespace=logging
```
   + connect a browser to `http://localhost:5601`
   + after GUI loads, click on the `Discover` button, the top icon on left side of page
   + change Index Pattern value to `logstash-*` and click `Next step`
   + click drop-down on `Configure Settings`, choose `@timestamp` and click `Create Index Pattern`
   + click on `Discover` button again to show log entries

## Viewing data

Useful fields to add are `log` and `kubernetes.pod_name`

A filter can be added to select only entries from specific pods. For example, to show log entries only from selected application pods, 
add a filter with `field=kubernetes.pod_name.keyword`, `operator=is one of`, select one or more pods, and then click `Save`.


## Warning about storage

By default Elastic Search will store all log data in its container and this storage will be in the root filesystem. Since it is unpleasant to fill root, if that much log data is expected look into configuring a volume mount for {{data-dir}} on a different filesystem.

## Removing log stack

`../scripts/logstack/stop-k3d-logstack.sh`
