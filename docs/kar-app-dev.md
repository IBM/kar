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

The KAR actor model is inherently developer friendly:
   + by default actors are single-threaded
   + actor-to-actor calls are analgous to subroutine calls, with simple addressing and argument passing
   + business logic can be easily moved between actors because remote actor targets are symbolic and persistent state relative
   + stdout and log tracing from different actor types can be kept separate or easily aggregated into fewer files
   + the persistent state of actors can be conveniently queried from the command line
   + a straightforward approach to implementing fault tolerant applications


The sections below address issues likely to be faced by KAR application developers,
and additional features KAR offers to assist in problem resolution.

## Local logging stack

Although actors are single threaded, using asynchronous messages a single method can 
generate lots of concurrent work in other actors.
Combining asynchronous messaging with the use of multiple KAR actor and web microservices 
increase the possibility that difficult to debug errors will occur.
Such errors will sometimes require examination of 
integrated log files from all components over extended time spans.

For developers that chose to work with kubernetes clusters in laptops or simple VM environments,
KAR includes scripts to deploy an Elasticsearch/Fluentd/Kibana logging stack along with 
any KAR application deployed in a K3D cluster.

See XXXY

## Local metrics stack

Another possible development issue is dealing with performance problems. 
The KAR application mesh transparently collects performance statistics on every actor method 
(AND WEB SERVER ENDPOINT???) running in application microserices.

For developers that chose to work with kubernetes clusters in laptops or simple VM environments,
KAR includes scripts to deploy a Prometheus/Grafana metrics stack to capture and
analyze these performance metrics for any KAR application deployed in a K3D cluster.
Some customization of the metrics stack is required for new applications;
the Reefer Container example application provides guidance in doing this.

See XYYY

## Fault tolerant design

The KAR approach to fault tolerant application design requires adherence to certain requirements.
For example, if an actor method is interrupted by an actor server fault, the method may be called
again with the same input arguments. 
KAR expects that the method be idempotent so that the final result is correct for the combination of whatever
was done on an interrupted call plus whatever was done on a repeated call.

If the interrupted method makes calls to other actors, the downstream actors might have to be made idempotent
as well.
Appropriate use of KAR tail calls can reduce or even eliminate the propagation of repeated calls beyond
the interrupted method itself.
Furthermore, the use of tail calls can significantly simplify checks in downstream methods because
they can be sure that the upstream method will only call when it has completed successfully.

The Reefer Container application includes fault generation scripts for randomly killing KAR application 
servers running in docker-compose or podman play kube, as well as for randomly killing K3D nodes containing
multiple KAR servers. See XYZZ








