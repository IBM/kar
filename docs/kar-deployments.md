# Overview

TODO - Dave to write

# Clusterless

TODO - Dave to write

# Kubernetes

TODO - Dave to write

# IBM Code Engine

TODO - Dave to write

# Hybrid Cloud

TODO - Dave to write


--------------

OLD GETTING STARTED TEXT TO BE UPDATED/REUSED.




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
the simplest clusterless local mode using our NodeJS-based examples.
After becoming familiar with this mode, either continue with running
some Java examples in the same clusterless mode or explore one of the
Kubernetes-based modes. 

# Prerequisites

1. Have an installation of Docker Desktop (Mac/Windows) or Docker Engine (Linux).

2.  You will need a local clone of the KAR git repository:
```shell
git clone git@github.ibm.com:solsa/kar.git
cd kar
```
Unless otherwise noted, all shell commands in this document assume
you are at the top-level directory of your local clone of the kar repo.

3. You will need the `kar` cli.  Currently you must build `kar` from
source, which requires a Go development environment to be available on
your machine. If you have Go installed, then build `kar` with
```shell
make cli
```

# Local Clusterless Deployment

## Deploying an instance of the KAR Runtime System

The KAR runtime system internally uses redis, kafka, and zookeeper.
You can deploy these as docker containers using docker-compose
by running:
```shell
./scripts/docker-composer.start.sh
```

After the script completes, configure your shell environment
to enable `kar` to access its runtime by doing
```shell
source ./scripts/kar-env-local.sh
```

## Run a NodeJS based example locally

### Prerequisites

You will need Node 12+ and NPM 6.12+ to run the NodeJS example.

###  Run a Hello World NodeJS Example

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

For more details on the NodeJS example, see its [README](../examples/service-hello-js/README.md).

## Run a Java based example locally

### Prerequisites

1. You will need Java 11 and Maven 3.6+ installed.

### Build the Java SDK and publish to your local .m2 repository

```shell
make installJavaSDK
```

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




# Local Cluster Deployment

Next, we will use [kind](https://kind.sigs.k8s.io/) to create a
virtual Kubernetes cluster using Docker on your development machine.
Using kind is only supported in KAR's "dev" mode where you will be
building your own docker images of the KAR system components and
pushing them into a local docker registry that we configure when
deploying kind.

## Prerequisites

1. You will need `kind` 0.9.0 installed locally.

2. You will need the `kubectl` cli installed locally.

3. You will need the `helm` (Helm 3) cli installed locally.

### Create your `kind` cluster and docker registry

```shell
./scripts/kind-start.sh
```

### Deploying the KAR Runtime System to the `kar-system` namespace

First, build the necessary docker images and push them to a local
registry that is accessible to kind with:
```shell
make dockerDev
```
Next, deploy KAR in dev mode by doing:
```shell
./scripts/kar-k8s-deploy.sh -dev
```

### Enable a namespace to run KAR-based applications.

**NOTE: We strongly recommend against enabling the `kar-system` namespace
  or any Kubernetes system namespace for KAR applications. Enabling
  KAR sidecar injection for these namespaces can cause instability.**

Enabling a namespace for deploying KAR-based applications requires
copying configuration secrets from the `kar-system` namespace and
labeling the namespace to enable KAR sidecar injection.  These steps
are automated by [kar-k8s-namespace-enable.sh](../scripts/kar-k8s-namespace-enable.sh).

The simplest approach is to KAR-enable the default namespace:
```shell
./scripts/kar-k8s-namespace-enable.sh default
```
The rest of our documentation assumes you will be deploying KAR
applications to the default namespace and omits the `-n <namespace>`
arguments to `kubectl` and `helm`.

If you are used to working with multiple Kubernetes namespaces,
you can use the same script to KAR-enable other namespaces.
If the namespace doesn't exist, the script will create it.
For example, to create and KAR-enable the `kar-apps` namespace execute:
```shell
./scripts/kar-k8s-namespace-enable.sh kar-apps
```

### Run a containerized example

Run the client and server as shown below:
```shell
$ cd examples/service-hello-js
$ kubectl apply -f deploy/server-dev.yaml
pod/hello-server created
$ kubectl get pods
NAME           READY   STATUS    RESTARTS   AGE
hello-server   2/2     Running   0          3s
$ kubectl apply -f deploy/client-dev.yaml
job.batch/hello-client created
$ kubectl logs jobs/hello-client -c client
Hello John Doe!
Hello John Doe!
$ kubectl logs hello-server -c server
Hello John Doe!
Hello John Doe!
$ kubectl delete -f deploy/client-dev.yaml
job.batch "hello-client" deleted
$ kubectl delete -f deploy/server-dev.yaml
pod "hello-server" deleted
```

## Deploying KAR on IBM Cloud Kubernetes Service

## Prerequisites

1. You will need an IKS cluster on which you have the cluster-admin role.

2. You will need the `kubectl` cli installed locally.

3. You will need the `helm` (Helm 3) cli installed locally.

## Deploy the KAR Runtime System

When deploying on IKS, you will use pre-built images from the KAR
project namespace in the IBM Cloud Container Registry.
Assuming you have set your kubectl context and have done an
`ibmcloud login` into the RIS IBM Research Shared account, you
can deploy KAR into your cluster in a single command:
```shell
./scripts/kar-k8s-deploy.sh
```

### Enable a namespace to run KAR-based applications.

```shell
./scripts/kar-k8s-namespace-enable.sh default
```

### Run a containerized example

Run the client and server as shown below:
```shell
$ cd examples/service-hello-js
$ kubectl apply -f deploy/server-icr.yaml
pod/hello-server created
$ kubectl get pods
NAME           READY   STATUS    RESTARTS   AGE
hello-server   2/2     Running   0          3s
$ kubectl apply -f deploy/client-icr.yaml
job.batch/hello-client created
$ kubectl logs jobs/hello-client -c client
Hello John Doe!
Hello John Doe!
$ kubectl logs hello-server -c server
Hello John Doe!
Hello John Doe!
$ kubectl delete -f deploy/client-icr.yaml
job.batch "hello-client" deleted
$ kubectl delete -f deploy/server-icr.yaml
pod "hello-server" deleted
```

# Next Steps

Now that you have run your first few KAR examples, you can continue to
explore in a number of directions.

1. Browse the examples and try running additional programs.
2. Experiment with mixing local and cluster mode executions.
3. Explore multi-cluster and hybrid application deployments by
   configuring the KAR Runtime system using `scripts/kar-env-ibmcloud.sh`
   to use EventStreams and Redis instances provisioned on the IBM
   Public cloud.
4. Deploy a KAR application on IBM Code Engine.
