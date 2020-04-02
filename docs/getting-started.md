# Getting Started with KAR

## Prerequisites

1. You need a Kubernetes 1.16 (or newer) cluster.
   a. You can use an IKS cluster without additional setup
   b. You can use [kind](https://kind.sigs.k8s.io/) to create a virtual Kubernetes cluster using Docker
      on your development machine. Just run [start-kind.sh](build/ci/start-kind.sh)
      to create a cluster.

2. You will need the `kubectl` cli installed locally.

3. You will need the `helm` (Helm 3) cli installed locally.

4. You will need a local git clone of this repository.

## Getting Started with KAR

### Deploying KAR to the `kar-system` namespace

Unless you are actively developing the KAR runtime, you will probably
want to deploy it using pre-built images from the KAR namespaces in
the IBM container registry (kar-dev, kar-stage, kar-prod). Currently,
you will have to ask Dave or Olivier for an apikey that enables this
access. After you have that apikey pass it as an argument to `kar-deploy.sh`
as shown below.
```script
./scripts/kar-deploy.sh -a <KAR_CR_APIKEY>
```

You can also deploy KAR in a dev mode where it will instead always
use locally built images for all KAR runtime components and examples.
Deploy this way with `./scripts/kar-deploy.sh -dev` and use
`make kindPushDev` to build and push images to your kind cluster.

### Enable a namespace to run KAR-based applications.

**NOTE: We strongly recommend against enabling the `kar-system` namespace
  or any Kubernetes system namespace for KAR applications. Enabling
  KAR sidecar injection for these namespaces can cause instability.**

Enabling a namespace for deploying KAR-based applications requires
copying configuration secrets from the `kar-system` namespace and
labeling the namespace to enable KAR sidecar injection.  These steps
are automated by
[kar-enable-namespace.sh](scripts/kar-enable-namespace.sh)

For example, to create and KAR-enable the `kar-apps` namespace execute:
```shell
./scripts/kar-enable-namespace.sh kar-apps
```

Or to KAR-enable an existing namespace, for example the `default`namespace:
```shell
./scripts/kar-enable-namespace.sh default
```

Now you are ready to run KAR applications!

### Running a sample application

First try running the incr example:
```shell
kubectl apply -f examples/incr/deploy/incr.yaml -n kar-apps
```
After a few seconds, you should see the following pods
```
(%) kubectl get pods -n kar-apps
NAME                           READY   STATUS    RESTARTS   AGE
incr-client-nrwst              1/2     Running   0          48m
incr-server-75944c4fc5-wd26m   2/2     Running   0          48m
```

Examining the logs of the client pod, you should see the number `43`.
```
(%) kubectl logs jobs/incr-client -c client -n kar-apps
43
```

To cleanup, do
```
kubectl delete -f examples/incr/deploy/incr.yaml -n kar-apps
```

### Running a sample application outside of Kubernetes

KAR also supports running applications outside of Kubernetes. Access
to Redis and Kafka is configured by defining environment variables
that are read by the `kar` executable that launches each application
process.

One simple way to experiment with this mode is to first deploy KAR on
kind as described above.  Then do `. ./scripts/kar-kind-env.sh` to
configure a shell to access Redis and Kafka.  Finally, run the `incr`
example by doing:
```shell
# start server
kar -app myApp -service myService node server.js &

# run client
kar -app myApp -service myClient node client.js
```
