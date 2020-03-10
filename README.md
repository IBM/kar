# KAR: Kubernetes Application Runtime

## Deploying KAR systems

Please follow the instructions in [deploying.md](docs/deploying.md).

## Running a sample application

After deploying KAR, you can run the incr example:
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

Examing the logs of the client pod, you should see the number `43`.
```
(%) kubectl logs jobs/incr-client -c client -n kar-apps
43
```

To cleanup, do
```
kubectl delete -f examples/incr/incr.yaml -n kar-apps
```
