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

Using KAR to build and run applications requires two things:
1. A running instance of the KAR runtime system.
2. The `kar` cli, which is used to launch application components and
   connect them to the KAR runtime system and each other in
   application service meshes that enable cross-component
   communication.

A running KAR application will be composed of one or more application
components. Individual components may be deployed as simple OS
processes running directly on a development machine or as containers
running inside one or more Kubernetes or OpenShift clusters. The
Kubernetes clusters may be local to the development machine (eg
Kubernetes in Docker Desktop, `kind` or `minishift`), or remote (eg
IBM Cloud Kubernetes Service or OpenShift). 

One of the values of KAR is that it enables developers to make an easy
and frictionless transition between these various modes, including
simultaneously running application components in multiple of these
modes for easier local debugging.

To simplify getting started with KAR, we suggest first starting with
the simplest clusterless local mode using a Node.js example.
After successfully running the first Node.js example, you can continue
by running additional Node.js and Java examples in the same
clusterless local mode.

Next, you can explore additional deployment modes such as deploying
KAR on a [Kubernetes cluster](kar-deployments.md#kubernetes-and-openshift),
on [IBM Code Engine](kar-deployments.md#ibm-code-engine),
or even spanning multiple execution environments in a
[Hybrid Cloud](kar-deployments.md#hybrid-cloud) deployment.

# Prerequisites

1. Have an installation of Docker Desktop (Mac/Windows) or Docker Engine (Linux).

2. Download the `kar` cli for your platform and the source release
   from the most recent KAR release at https://github.com/ibm/kar/releases.

3. Put the `kar` cli binary on your path.

4. Unzip/untar the source release.

Unless otherwise noted, all shell commands in this document assume you
are at the top-level directory of the KAR source release.

To ensure the instructions are compatible with your version of KAR,
always consult the `docs/getting-started.md` in your local copy; not
the online version which tracks the tip of the KAR `main` branch.

# Local Clusterless Deployment

## Deploying an instance of the KAR Runtime System

The KAR runtime system internally uses Redis as a persistent store and
Kafka as a reliable message transport (Kafka internally uses ZooKeeper
for distributed consensus).  You can deploy these
dependencies as docker containers using docker-compose by running:
```shell
RESTART_POLICY=always ./scripts/docker-compose-start.sh
```

After the script completes, configure your shell environment
to enable `kar` to access its runtime by doing
```shell
source ./scripts/kar-env-local.sh
```

## Run a Node.js based example locally

### Prerequisites

You will need Node.js 12+ and NPM 6.12+ to run the Node.js example.

###  Run a Hello World Node.js Example

In one window:
```shell
source scripts/kar-env-local.sh
cd examples/service-hello-js
npm install --prod 
kar run -app hello-js -service greeter node server.js
```

In a second window:
```shell
source scripts/kar-env-local.sh
cd examples/service-hello-js
kar run -app hello-js node client.js
```

You should see output like shown below in both windows:
```
2020/04/02 17:41:23 [STDOUT] Hello John Doe!
2020/04/02 17:41:23 [STDOUT] Hello John Doe!
```
The client process will exit, while the server remains running. You
can send another request, or exit the server with a Control-C.

You can also use the `kar` cli to invoke the service directly:
```shell
kar rest -app hello-js post greeter helloJson '{"name": "Alan Turing"}'
```

For more details on the Node.js example, see its [README](../examples/service-hello-js/README.md).

## Run a Java based example locally

### Prerequisites

1. You will need Java 11 and Maven 3.6+ installed.

### Run the Hello World Java Example

In one window:
```shell
source scripts/kar-env-local.sh
cd examples/service-hello-java
mvn package
kar run -app hello-java -service greeter mvn liberty:run
```

In a second window, run the Java client program
```shell
source scripts/kar-env-local.sh
cd examples/service-hello-java
kar run -app hello-java java -jar client/target/kar-hello-client-jar-with-dependencies.jar
```

You should see output like shown below in both windows:
```
2020/10/02 14:56:23.770749 [STDOUT] Hello Gandalf the Grey
```
The client process will exit, while the server remains running. You
can send another request, or exit the server with a Control-C.

You can also use the `kar` cli to invoke the service directly:
```shell
kar rest -app hello-java post greeter helloJson '{"name": "Alan Turing"}'
```

For more details on the Java example, see its [README](../examples/service-hello-java/README.md).


## Undeploying an instance of the KAR Runtime System

Undeploying your local instance of the KAR runtime system entails stopping
and removing the docker containers for Redis, Kafka, and ZooKeeper. Note that
this will also remove all saved state for any KAR-based applications you have run. 
Undeploy these containers using docker-compose by running:
```shell
./scripts/docker-compose-stop.sh
```

# Next Steps

Now that you have run your first few KAR examples, you can continue to
explore in a number of directions.

1. Browse the examples and try running additional programs.
2. Explore [additional deployment options](kar-deployments.md)
   for KAR including [Kubernetes](kar-deployments.md#kubernetes-and-openshift),
   [IBM Code Engine](kar-deployments.md#ibm-code-engine),
   and [Hybrid Cloud](kar-deployments.md#hybrid-cloud) deployments.
