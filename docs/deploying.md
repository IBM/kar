# Deploying KAR on Kubernetes

## Prepare the `kar-system` namespace

Before deploying KAR for the first time on a cluster, you will need to
create and configure the `kar-system` namespace.  This namespace will
be used to execute KAR system components.

Perform the following operations:
1. Create the namespace
```shell
kubectl create ns kar-system
```

2. Create an image pull secret that allows access to the KAR
namespaces in the IBM container registry (kar-dev, kar-stage,
kar-prod). Currently, you will need to ask Dave or Olivier for an
apikey that enables read-only access. After you have <APIKEY> execute
the command below (replacing <APIKEY> with the actual value).

```shell
kubectl --namespace kar-system create secret docker-registry kar.ibm.com.image-pull --docker-server=us.icr.io --docker-username=iamapikey --docker-email=kar@ibm.com --docker-password=<APIKEY>
```

## Install the KAR Helm chart
From the top-level of a git clone of the KAR repo.

```shell
helm install kar charts/kar -n kar-system
```

## Enable namespaces for KAR-based applications.

**NOTE: We strongly recommend against enabling the `kar-system` namespace
  or any Kubernetes system namespace for KAR applications. Enabling
  KAR sidecar injection for these namespaces can cause instability.**

For every namespace in which you want to deploy KAR-based
applications, you will need to perform the following operations. In
the sample commands below, we will enable the `kar-apps` namespace.

1. Create the namespace if it doesn't already exist.
```shell
kubectl create ns kar-apps
```

2. Copy the `kar.ibm.com.image-pull` secret from `kar-system` to the new namespace.
```shell
kubectl get secret kar.ibm.com.image-pull -n kar-system -o yaml | sed 's/kar-system/kar-apps/g' | kubectl -n kar-apps create -f -
```

3. Label the namespace to enable the KAR sidecar injector.
```shell
kubectl label namespace kar-apps kar.ibm.com/enabled=true
```

