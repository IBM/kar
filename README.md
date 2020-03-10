# KAR: Kubernetes Application Runtime

## Getting Started with KAR

### Deploy KAR to the `kar-system` namespace

Please follow the instructions in [deploying.md](docs/deploying.md).

### Create a namespace in which to run KAR-based applications.

**NOTE: We strongly recommend against enabling the `kar-system` namespace
  or any Kubernetes system namespace for KAR applications. Enabling
  KAR sidecar injection for these namespaces can cause instability.**

Enabling a namespace for deploying KAR-based applications requires
copying configuration secrets from the `kar-system` namespace and
labeling the namespace to enable KAR sidecar injection.  These steps
are automated by
[create-app-namespace.sh](scripts/create-app-namespace.sh)

```shell
./scripts/create-app-namespace.sh kar-apps
```

Now you are ready to run KAR applications!

### Running a sample application

First try running the incr example:
```shell
kubectl apply -f examples/incr/incr.yaml -n kar-apps
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
kubectl delete -f examples/incr/incr.yaml -n kar-apps
```
