## Yorktown site simulation

An agent-based simulation of activity in the Yorktown site.  Researchers
arrive at work, move around the site, drink coffee, and have meetings.

The simulation is designed to enable load testing of the actor runtime
system and show how actors can interact with pub/sub.

The actors are:
+ Each Researcher is an actor. Researchers are the active agents that drive the simulation.
+ Each of the 7,680 (3 x 40 x 64) office locations in Yorktown is represented by an actor that tracks it current occupancy.
+ Each Floor in the site is represented by an actor (to track population by floor)
+ The Site itself is an actor (used for simulation control and reporting)

When site reporting is active, the Site actor will periodically
publish occupancy stats on the `siteReport` channel.  Interested
processes can subscribe to `siteReport` to track the simulation.

## Running locally

The first time you run locally, you will need to execute `npm install`.

The easiest way to run locally is to invoke the script `./deploy/runServerLocally.sh` in one shell terminal and `./deploy/runClientLocally.sh` in another terminal. This will simulate one work day in approximately a minute.  Typical output is shown below:
```shell
Daves-MacBook-Pro:actors-ykt dgrove$ ./deploy/runClientLocally.sh 
2020/04/03 15:18:17 [WARNING] starting...
2020/04/03 15:18:17 [INFO] KAR_PORT=30666 KAR_APP_PORT=8080
2020/04/03 15:18:17 [INFO] launching service...
2020/04/03 15:18:18 [INFO] no sidecar for actor type Site, retrying
2020/04/03 15:18:18 [INFO] generation 10, sidecar 91d54558-9b13-4c6f-8faf-4b2203608471, claims [0]
2020/04/03 15:18:18 [INFO] increasing partition count to 2
2020/04/03 15:18:18 [INFO] no sidecar for actor type Site, retrying
2020/04/03 15:18:19 [INFO] no sidecar for actor type Site, retrying
2020/04/03 15:18:20 [INFO] generation 11, sidecar 91d54558-9b13-4c6f-8faf-4b2203608471, claims [0]
2020/04/03 15:18:20 [STDOUT] Staring YKT simulation: {"workers":10,"thinkms":2000,"steps":20}
2020/04/03 15:18:26 [STDOUT] Num working is 10
2020/04/03 15:18:31 [STDOUT] Num working is 10
2020/04/03 15:18:36 [STDOUT] Num working is 10
2020/04/03 15:18:41 [STDOUT] Num working is 8
2020/04/03 15:18:46 [STDOUT] Num working is 3
2020/04/03 15:18:51 [STDOUT] Num working is 1
2020/04/03 15:18:56 [STDOUT] Num working is 0
2020/04/03 15:18:56 [STDOUT] { bucketSizeInMS: 100, delayHistogram: [ 185, 15 ] }
2020/04/03 15:18:56 [STDOUT] SUCCESS
2020/04/03 15:18:56 [STDOUT] Requesting server shutdown
2020/04/03 15:18:56 [STDOUT] Terminating sidecar
2020/04/03 15:18:56 [INFO] Invoking cancel() in response to kill request
2020/04/03 15:18:56 [INFO] service exited normally
2020/04/03 15:18:56 [WARNING] exiting...
```

```shell
Daves-MacBook-Pro:actors-ykt dgrove$ ./deploy/runServerLocally.sh 
2020/04/03 15:17:56 [WARNING] starting...
2020/04/03 15:17:57 [INFO] KAR_PORT=62430 KAR_APP_PORT=8080
2020/04/03 15:17:57 [INFO] launching service...
2020/04/03 15:17:57 [INFO] generation 9, sidecar 317aa735-8662-4c18-a1a3-0c30e9a4b956, claims [0]
2020/04/03 15:18:18 [INFO] generation 10, sidecar 317aa735-8662-4c18-a1a3-0c30e9a4b956, claims []
2020/04/03 15:18:18 [INFO] increasing partition count to 2
2020/04/03 15:18:20 [INFO] generation 11, sidecar 317aa735-8662-4c18-a1a3-0c30e9a4b956, claims [1]
2020/04/03 15:18:20 [STDOUT] {
2020/04/03 15:18:20 [STDOUT]   totalWorking: 0,
2020/04/03 15:18:20 [STDOUT]   floor0: 0,
2020/04/03 15:18:20 [STDOUT]   floor1: 0,
2020/04/03 15:18:20 [STDOUT]   floor2: 0,
2020/04/03 15:18:20 [STDOUT]   coffee: 0,
2020/04/03 15:18:20 [STDOUT]   cafeteria: 0,
2020/04/03 15:18:20 [STDOUT]   time: 'Fri Apr 03 2020 15:18:20 GMT-0400 (Eastern Daylight Time)'
2020/04/03 15:18:20 [STDOUT] }
2020/04/03 15:18:20 [STDOUT] 10 starting their shift of 20 tasks at ykt
2020/04/03 15:18:21 [STDOUT] 0 entered Site ykt
2020/04/03 15:18:21 [STDOUT] 3 entered Site ykt
2020/04/03 15:18:21 [STDOUT] 2 entered Site ykt
2020/04/03 15:18:21 [STDOUT] 1 entered Site ykt
2020/04/03 15:18:21 [STDOUT] 4 entered Site ykt
2020/04/03 15:18:21 [STDOUT] 7 entered Site ykt
2020/04/03 15:18:21 [STDOUT] 5 entered Site ykt
2020/04/03 15:18:21 [STDOUT] 6 entered Site ykt
2020/04/03 15:18:21 [STDOUT] 9 entered Site ykt
2020/04/03 15:18:21 [STDOUT] 8 entered Site ykt
2020/04/03 15:18:22 [STDOUT] Coffee time for 1
2020/04/03 15:18:22 [STDOUT] Coffee time for 1
.
.
...many more messages...
.
.
2020/04/03 15:18:51 [STDOUT] 0 left Site ykt
2020/04/03 15:18:51 [STDOUT] Quitting time for 0
2020/04/03 15:18:56 [STDOUT] {
2020/04/03 15:18:56 [STDOUT]   totalWorking: 0,
2020/04/03 15:18:56 [STDOUT]   floor0: 0,
2020/04/03 15:18:56 [STDOUT]   floor1: 0,
2020/04/03 15:18:56 [STDOUT]   floor2: 0,
2020/04/03 15:18:56 [STDOUT]   coffee: 0,
2020/04/03 15:18:56 [STDOUT]   cafeteria: 0,
2020/04/03 15:18:56 [STDOUT]   time: 'Fri Apr 03 2020 15:18:56 GMT-0400 (Eastern Daylight Time)'
2020/04/03 15:18:56 [STDOUT] }
2020/04/03 15:18:56 [STDOUT] <100ms	185
2020/04/03 15:18:56 [STDOUT] <200ms	15
2020/04/03 15:18:56 [STDOUT] Shutting down service
2020/04/03 15:18:56 [INFO] Invoking cancel() in response to kill request
2020/04/03 15:19:01 [INFO] service exited normally
2020/04/03 15:19:01 [WARNING] exiting...
```

## Running in Kubernetes

There is a Helm chart to deploy the simulation on Kubernetes.  By
default it creates a `Deployment` with two replicas to execute the
simulation.  The client is configured as a Helm test that initiates a
single workday and verifies that the expected work was completed.

```shell
$ helm install ykt ./deploy/chart --set image=example-ykt:dev
NAME: ykt
LAST DEPLOYED: Fri Apr  3 16:08:35 2020
NAMESPACE: default
STATUS: deployed
REVISION: 1
$ kubectl get pods 
NAME                          READY   STATUS        RESTARTS   AGE
ykt-server-5cdb4699d9-5s4cc   2/2     Running       0          3s
ykt-server-5cdb4699d9-bqmc9   2/2     Running       0          3s
$ helm test ykt 
Pod ykt-client pending
Pod ykt-client pending
Pod ykt-client running
Pod ykt-client succeeded
NAME: ykt
LAST DEPLOYED: Fri Apr  3 16:08:35 2020
NAMESPACE: default
STATUS: deployed
REVISION: 1
TEST SUITE:     ykt-client
Last Started:   Fri Apr  3 16:08:45 2020
Last Completed: Fri Apr  3 16:09:21 2020
Phase:          Succeeded
```
