# Using KAR with the IBM Cloud

## Prerequisites

To use IBM Cloud compute services (IKS, ROKS, CodeEngine) to execute
KAR application components, you must first provision a set of managed
services on the IBM Public Cloud and the IAM apikeys to access them.

### Event Streams

You will need a Standard EventStreams instance.  Once it is allocated,
create a service credential to access it.

TODO: Detailed instructons.

### Database for Redis

You will need a Database for Redis instance.  Once it is allocated,
create a service credential to access it, using the same name as you
used for the EventStreams service credential.

TODO: Detailed instructions.

### IBM Cloud Container Registry Access

You will need an IBM Cloud Container Registry namespace and an apikey
that enables read access to that namespace.

TODO: Detailed instructions.

## Using KAR on IKS

You will need an IKS cluster on which you have the cluster-admin role.

### Install the KAR runtime system on your IKS cluster

```shell
./scripts/kar-k8s-deploy.sh -m <servicekey> -c <icr api key>
```

### Enable a namespace for KAR applications
```shell
./scripts/kar-k8s-namespace-enable.sh default
```

### Deploy applications

You can now deploy applications using helm or `kubectl` by creating
Kubernetes deployments that are annotated to inject the `kar` sidecar
container into its pods.  See the examples.

To run in a hybrid mode, where some components are on IKS and some
components are on your laptop, configure your local environment with:
```shell
source scripts/kar-env-ibmcloud.sh <service-key>
```

## Using KAR on IBM Code Engine

### Create a Code Engine project
```shell
ibmcloud ce project create --name kar-project
```

### Configure the project for KAR
```shell
./scripts/kar-code-engine-project-enable.sh <service-key> <cr-apikey>
```

### Deploy a KAR component

Although this can be done directly with the `ibmcloud ce` cli, it
requires a fairly extensive set of command line arguments.  The script
`kar-ce-run.sh` wraps `ibmcloud ce` to simplify the process.

```shell
./scripts/kar-ce-run.sh -app hello -image us.icr.io/research/kar-dev/examples/js/service-hello -name hello-js-server -service greeter
```
