# Prerequisites

1. You need a Kubernetes 1.16 (or newer) cluster.
   a. You can use an IKS cluster provisioned on the IBM Public Cloud.
   b. You can use [kind](https://kind.sigs.k8s.io/) to create a virtual Kubernetes cluster using Docker
      on your development machine. Just run [start-kind.sh](../build/ci/start-kind.sh)
      to create a virtual cluster.

2. You will need the `kubectl` cli installed locally.

3. You will need the `helm` (Helm 3) cli installed locally.

# Getting Started with KAR

In the sections below, the sample commands are meant to be executed in
the top level directory of a local git clone of this repository. Get
one by doing:
```script
git clone git@github.ibm.com:solsa/kar.git
cd kar
```

## Deploying KAR to the `kar-system` namespace

You can deploy KAR using pre-built images from KAR project namespaces
in the IBM container registry (kar-dev, kar-stage,
kar-prod). Currently, you will have to ask Dave or Olivier for an
apikey to access this namespace.  Use the apikey as an argument to
`kar-deploy.sh`:
```script
./scripts/kar-deploy.sh -a <KAR_CR_APIKEY>
```

Alternatively, you can deploy in dev mode where KAR will use
locally built images for all KAR runtime components and examples.
Since it is bypassing the container registry, this mode is preferred
for local development, but only works with `kind`, not IKS.
Deploy KAR in dev mode by doing:
```shell
make kindPushDev
./scripts/kar-deploy.sh -dev
```

## Enable a namespace to run KAR-based applications.

**NOTE: We strongly recommend against enabling the `kar-system` namespace
  or any Kubernetes system namespace for KAR applications. Enabling
  KAR sidecar injection for these namespaces can cause instability.**

Enabling a namespace for deploying KAR-based applications requires
copying configuration secrets from the `kar-system` namespace and
labeling the namespace to enable KAR sidecar injection.  These steps
are automated by
[kar-enable-namespace.sh](../scripts/kar-enable-namespace.sh)

For example, to create and KAR-enable the `kar-apps` namespace execute:
```shell
./scripts/kar-enable-namespace.sh kar-apps
```

Or to KAR-enable an existing namespace, for example the `default`namespace:
```shell
./scripts/kar-enable-namespace.sh default
```

Now you are ready to run KAR applications!

## Running KAR-based applications

To demonstrate the different modes of running KAR applications, we'll
use a simple greeting server.  A server process receives a request
from a client containing a name and responds to each request with a
greeting.

### Mode 1: running completely inside Kubernetes

TODO: fill in commands

### Mode 2: running completely locally

TODO: fill in commands

### Mode 3: run the server in Kubernetes and the client locally

TODO: fill in commands

