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

# Overview

This document describes the different deployment modes supported by
KAR, including:
   + [Clusterless](#clusterless)
   + [Kubernetes and OpenShift](#kubernetes-and-openshift)
   + [IBM Code Engine](#ibm-code-engine)
   + [Hybrid Cloud](#hybrid-cloud)

## KAR Runtime System Components

The KAR runtime system internally uses Redis as a persistent store and
Kafka as a reliable message transport (Kafka internally uses ZooKeeper
for distributed consensus). The Redis and Kafka instances must be reachable
by every `kar` runtime process in order for them to operate correctly and
form the application service mesh.

The Redis and Kafka instances can be provided in multiple ways, each
supporting different scenarios:
   + They can be run locally as Docker containers, supporting a local
     clusterless mode which is suitable for development.
   + They can be run as internally-accessible services/deployments on a
     Kubernetes or OpenShift cluster, supporting the deployment of
     KAR applications within that cluster.
   + They can be run as externally-accessible services/deployments on a
     Kubernetes or OpenShift cluster, supporting the deployment of
     KAR applications both inside and outside that cluster.
   + They can be provided as cloud managed services, supporting the
     deployment of KAR applications across multiple execution engines
     including Kubernetes and OpenShift clusters, IBM Code Engine,
     edge computing devices, and developer laptops.
Depending on the scenario, Redis and Kafka may use clustered
configurations to support high availability and increased scalability.

When deployed on a Kubernetes of OpenShift cluster, the KAR runtime
system also includes a mutating web hook that supports
injecting a "sidecar" container into Pods that are annotated as
containing KAR application components.  This significantly simplifies
the configuration of these components by automating the injection of
the credentials needed to connect to the Redis and Kafka instances
being used by KAR.

## Prerequisites

Throughout this document, we assume that all of the
[prerequisites](getting-started.md#prerequisites) outlined in the
getting started document have been met. We also assume that if you are
deploying to a Kubernetes cluster or using other public cloud managed
services in your deployment, that you have the necessary clis and
tools already installed and have some familiarity with using them.

# Clusterless

The Clusterless deployment mode runs Redis, Kafka, and ZooKeeper
as docker containers on your local machine. Your application components
and the KAR service mesh all run as local processes on your machine.

## Deployment

Deploy Redis, Kafka, and ZooKeeper using docker-compose by running:
```shell
./scripts/docker-composer.start.sh
```

After the script completes, configure your shell environment
to enable `kar` to access Redis and Kafka by doing
```shell
source ./scripts/kar-env-local.sh
```

## Run a Hello World Node.js Example

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

## Undeploying

Undeploy the Redis, Kafka, and ZooKeeper containers using
docker-compose by running:
```shell
./scripts/docker-composer.stop.sh
```

# Kubernetes and OpenShift

This section covers the base deployment scenario where all the
components of the KAR application will be realized as Pods running
within a single cluster. In this scenario, Redis, Kafka, and ZooKeeper
are also deployed as Pods running within the cluster. For scenarios
including multiple clusters and/or applications that are split between a
cluster and edge devices (or developer laptops) see [Hybrid
Cloud](#hybrid-cloud) below.

## General Overview

This section outlines some general principles that are true of
any in-cluster deployment of KAR.

For its in-cluster configurations, the KAR runtime system is deployed
in the `kar-system` namespace and includes a mutating webhook whose
job is to inject a "sidecar" container containing the `kar` executable
into every Pod that is annotated with `kar.ibm.com/app`. This
machinery enables existing Helm charts and Kubernetes YAML to be
adapted for KAR with minimal changes.  The mutating webhook process
the following annotations:
   + kar.ibm.com/app - sets the `-app` argument of `kar run`
   + kar.ibm.com/actors: sets the `-actors` argument of `kar run`
   + kar.ibm.com/service: sets the `-service` argument of `kar run`
   + kar.ibm.com/verbose: sets the `-verbose` argument of `kar run`
   + kar.ibm.com/appPort: sets the `-app_port` argument of `kar run`
   + kar.ibm.com/runtimePort: sets the `-runtime_port` argument of `kar run`
   + kar.ibm.com/extraArgs: additional command line arguments for `kar run`

If you are using a release version of the `kar` cli then, by default,
the matching KAR runtime images will be pulled from our public quay.io
image repository. If you have built your own `kar` cli from source then,
by default, the KAR runtime images will be pulled from your local image
repository that is expected to be running at `localhost:5000`. It is also
possible to configure KAR to pull its runtime images from a non-local
private registry; this results in an additional `kar.ibm.com.image-pull`
secret being created.

After the KAR runtime system is successfully deployed to the
`kar-system` namespace, you can enable other namespaces
to host KAR applications. This enablement entails labeling the namespace
with `kar.ibm.com/enabled=true` and replicating the
`kar.ibm.com.runtime-config` secret and, optionally, the `kar.ibm.com.image-pull`
secret in the namespace. The base installation script
automatically enables the `default` namespace for KAR applications.
To enable additional namespaces, you can use the script
[kar-k8s-namespace-enable.sh](../scripts/kar-k8s-namespace-enable.sh).

Once a namespace is thus enabled, you can deploy KAR application components to the
namespace using Helm or kubectl by adding the annotations described above.

**NOTE: We strongly recommend against enabling the `kar-system` namespace
  or any Kubernetes system namespace for KAR applications. Enabling
  KAR sidecar injection for these namespaces can cause instability.**

## Deploying on an IBM Cloud Kubernetes or OpenShift cluster

You will need a cluster on which you have the cluster-admin role.

### Deploying the KAR Runtime System to the `kar-system` namespace

Assuming you have set your kubectl context, you
can deploy KAR into your cluster in a single command:
```shell
./scripts/kar-k8s-deploy.sh
```

### Run a containerized example

Run the client and server as shown below:
```shell
$ cd examples/service-hello-js
$ kubectl apply -f deploy/server-quay.yaml
pod/hello-server created
$ kubectl get pods
NAME           READY   STATUS    RESTARTS   AGE
hello-server   2/2     Running   0          3s
$ kubectl apply -f deploy/client-quay.yaml
job.batch/hello-client created
$ kubectl logs jobs/hello-client -c client
Hello John Doe!
Hello John Doe!
$ kubectl logs hello-server -c server
Hello John Doe!
Hello John Doe!
$ kubectl delete -f deploy/client-quay.yaml
job.batch "hello-client" deleted
$ kubectl delete -f deploy/server-quay.yaml
pod "hello-server" deleted
```

If you have built your own docker images and pushed
them to your `localhost:5000 registry`,
you use them by doing:
```
$ kubectl apply -f deploy/server.yaml
$ kubectl apply -f deploy/client.yaml
```

### Undeploying

You can disable a specific namespace for KAR applications by running
```shell
./scripts/kar-k8s-namespace-disable.sh <namespace>
```

You can undeploy KAR entirely by running
```shell
./scripts/kar-k8s-undeploy.sh
```

## Deploying on a local Kubernetes cluster

For ease of development, it can be convenient to deploy KAR to a
Kubernetes or OpenShift cluster running on your local development
machine.  Several options exist for creating a local cluster; we
describe them next. In all cases, we recommend also running a local
docker registry to enable Kubernetes to pull images without requiring
you to push your development images to an external docker registry.

### Start your local cluster.

#### Docker Desktop

If you are using Docker Desktop on MacOS or Windows, you
can enable a built-in Kubernetes cluster by checking a box in the UI.

#### Kind

We can use [kind](https://kind.sigs.k8s.io/) to create a
virtual Kubernetes cluster using Docker on your development machine.

You will need `kind` 0.10.0 installed locally.

KAR requires specific configuration of `kind`.  We have automated
this in a script.
```shell
./scripts/kind-start.sh
```
#### Rancher K3s

Follow the directions to install [K3s](https://rancher.com/docs/k3s/latest/en/quick-start/).

To enable a separately installed kubectl to access this cluster:
```shell
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
```

Although K3s must be started and stopped by root, it is possible to enable kubectl access from a specific non-root `userID` by doing:
```shell
sudo chown userID /etc/rancher/k3s/k3s.yaml
```

### Start local docker registry

The rest of our instructions assume you will run a local docker
registry on `localhost:5000`.  To ensure one is running, execute
```shell
./scripts/start-local-registry.sh
```

### Build and locally publish images

Next, build the necessary docker images and push them to a local
registry.
```shell
make docker
```

When you rebuild a KAR docker image and want it to be accessible to
your Kubernetes cluster, you will need to either do `make docker`
or individually `docker push` the image to the localhost:5000 registry.

### Deploying the KAR Runtime System to the `kar-system` namespace

Next, deploy KAR in dev mode by doing:
```shell
./scripts/kar-k8s-deploy.sh
```

#### Run a containerized example

Run the client and server as shown below:
```shell
$ cd examples/service-hello-js
$ kubectl apply -f deploy/server.yaml
pod/hello-server created
$ kubectl get pods
NAME           READY   STATUS    RESTARTS   AGE
hello-server   2/2     Running   0          3s
$ kubectl apply -f deploy/client.yaml
job.batch/hello-client created
$ kubectl logs jobs/hello-client -c client
Hello John Doe!
Hello John Doe!
$ kubectl logs hello-server -c server
Hello John Doe!
Hello John Doe!
$ kubectl delete -f deploy/client.yaml
job.batch "hello-client" deleted
$ kubectl delete -f deploy/server.yaml
pod "hello-server" deleted
```

### Undeploying

You can disable a specific namespace for KAR applications by running
```shell
./scripts/kar-k8s-namespace-disable.sh <namespace>
```

You can undeploy KAR entirely with
```shell
./scripts/kar-k8s-undeploy.sh
```

# IBM Code Engine

IBM Code Engine is a multi-tenant Knative service provided by the IBM
Public Cloud. We can use IBM Code Engine as the compute engine for KAR
applications by deploying components as Code Engine applications (aka
Knative services).

We will not deploy Redis and Kafka on Code Engine; instead we will
provision instances of Database for Redis and EventStreams in the same
IBM Public Cloud region as the Code Engine service we are using to run
the application components. We will configure the Code Engine project
to enable the `kar` runtime processes to connect to these instances.

To simplify the flow of deploying on Code Engine, KAR application containers
intended for Code Engine deployment need to contain both the
application itself and the `kar` cli. We configure these containers to
execute in "sidecar in container" mode. If you are using a container
derived from the KAR Java or JavaScript SDK base images, this can be
done simply by setting the `KAR_SIDECAR_IN_CONTAINER` environment
variable.

There is currently no integration between Code Engine's autoscaling
capabilities and the Kafka topics that indicate the actual application
load. Therefore we currently bypass Code Engine's autoscaler and
deploy with a fixed number of containers for each application
component.

## Deployment

### Managed Services

Use the IBM Cloud Console to create resources.  Please consult the
documentation for each managed service if you need detailed
instructions.

You will need a Standard EventStreams instance.  Once it is allocated,
create a service credential to access it.

You will need a Database for Redis instance.  Once it is allocated,
create a service credential to access it, using the same name as you
used for the EventStreams service credential.

### Code Engine Project

Create a Code Engine project
```shell
ibmcloud ce project create --name kar-project
```

Then, configure the project for KAR by creating the
`kar.ibm.com.runtime-config` secret.
This step is automated by a script that takes the
service credential name and
uses the `ibmcloud` cli to extract information and create the secrets.
```shell
./scripts/kar-ce-project-enable.sh <service-credential>
```

### Optionally configure your local environment

Because we are using a Redis and Kafka instance that are accessible
both to containers running in IBM Code Engine and to your laptop, we
have the option of deploying applications with some components running
on the cloud in IBM Code Engine and others running locally. To enable
this option, you need to setup your local environment so that `kar`
can connect to your public cloud Redis and EventStreams instances.
Do this by running
```shell
source scripts/kar-env-ibmcloud.sh <service-credential>
```

## Run a Hello World Node.js Example

Although deploying a KAR application component to Code Engine can can
be done directly with the `ibmcloud ce` cli, it requires a fairly
extensive set of command line arguments.  The script `kar-ce-run.sh`
wraps `ibmcloud ce` to simplify the process. It automatically targets
the current Code Engine project (change the targeted project with
`ibmcloud ce project target <project-name>`).
```shell
./scripts/kar-ce-run.sh -app hello -image quay.io/ibm/kar-examples-js-service-hello -name hello-js-server -service greeter
```

Once the server component is deployed, you can use the `kar` cli to
invoke the service directly:
```shell
kar rest -app hello post greeter helloJson '{"name": "Alan Turing"}'
```

You've just run your first hybrid cloud application that uses KAR to
connect components running on your laptop (an "edge device") and the
IBM Public Cloud into a unified application mesh!

## Undeploying

You can undeploy an application component with
```shell
ibmcloud ce application delete --name hello-js-server
```

You can disable a Code Engine project for KAR applications with
```shell
./scripts/kar-ce-project-disable.sh kar-project
```
or delete it entirely with
```shell
ibmcloud ce project delete --name kar-project
```

# Hybrid Cloud

The key to a Hybrid Cloud deployment of KAR is to provision a Redis
and Kafka instance that are accessible to all of the compute elements
you want to utilize.  This can include Kubernetes clusters, edge
devices, virtual machines, development laptops, and managed compute
services such as Code Engine.

In general, you need to first provision the Redis and Kafka instances
and then in each computing environment create the configuration
information that enables `kar` to access them.  In Kubernetes and
OpenShift clusters and in IBM Code Engine, this means creating the
`kar.ibm.com.runtime-config` secret. For local environments or VMs
this means setting a collection of `KAFKA_` and `REDIS_` environment
variables.

## Using the IBM Public Cloud

### Provision Managed Services

Use the IBM Cloud Console to create resources.  Please consult the
documentation for each managed service if you need detailed
instructions.

You will need a Standard EventStreams instance.  Once it is allocated,
create a service credential to access it.

You will need a Database for Redis instance.  Once it is allocated,
create a service credential to access it, using the same name as you
used for the EventStreams service credential.

### Configuring compute engines

#### Kubernetes or OpenShift clusters

Install the KAR runtime system on your IKS cluster
```shell
./scripts/kar-k8s-deploy.sh -m <service-credential>
```

#### Code Engine

Enable a project for KAR applications
```shell
./scripts/kar-ce-project-enable.sh <service-credential>
```

#### Local environment or VMs
Set the necessary `KAFKA_` and `REDIS_` environment variables with
```shell
source scripts/kar-env-ibmcloud.sh <service-credential>
```

### Deploying Applications

Deploy each application component to the desired compute engine using
the scripts/tooling appropriate for that engine as described elsewhere
in this document.
