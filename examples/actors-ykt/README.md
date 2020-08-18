## IBM Research site simulation

An agent-based simulation of daily activity in the Research division.
Researchers commute to and from work, move around their assigned site,
drink coffee, have meetings, and otherwise fill their work day.

The simulation is designed to enable load testing of the actor runtime
system and to illustrate how to express several common application
patterns using KAR's actor programming model.

The entities in the simulation are:
+ Researcher
   + Researchers are the active agents that drive the simulation. 
     They are hired by a Company to work in a Site for a specific
     number of days.
   + As is common in simulations, Researchers are invoked by
     time-triggered `reminder` events to execute a simulation step.
     To support load testing and performance analysis, the application
     collects statistics on the delay between the time a reminder
     was scheduled to be fired and when it was actually delivered to
     its target actor. Each Researcher is expected to complete a
     known number of moves during the "career"; these numbers can be
     externally verified to test KAR's ability to support fault-tolerance.
   + The Researcher's `move` method is also a good summary of how to use
     the various pieces of the KAR actor model.  It uses both the
     synchronous `call` and asynchronous `tell` APIs to interact with
     other actors.  It uses `schedule` to re-invoke itself for
     the next step of the simulation. It uses `setMultiple` to checkpoint
     its own state at an application-chosen point to enable the simulation
     to recover from failures with a consistent view of its progress.
+ Office
   + Each Office represents a physical office in a Site.
   + Offices keep track of their current occupants, which is used by the
     simulation logic to determine if two Researchers will interact in
     in their current `move`.
   + Each Site contains 7,682 Offices, virtually all of which are empty
     at any given time.
   + Offices are designed to stress the ability of the KAR actor runtime
     system to automatically passivate inactive actor instances to stable
     storage to optimize memory usage.
+ Site
   + A Site represents a physical location that contains Offices.
   + There are a small number of Site instances in the simulation.
   + Every time a Researcher changes their activity, they use `tell` to
     report what they did to their Site. Thus, as the number of 
     Researchers is ramped up, the statistics gathered by the
     Site on the observed latency of the delivery of these messages
     can be used to assess KAR's ability to deliver
     large volumes of asynchronous messages to a small number
     of actor instances.
   + Sites aggregate statistics and periodically publish this data as
     events from the `siteReport` method.
   + The Site class also illustrates how an optimistic fault tolerance
     can be implemented in KAR.  Critical, but slowly changing data
     (the reminder delays collected in `retire`) is eagerly checkpointed.
     Non critical and rapidly changing data (the current state of every
     Site employee) is only checkpointed during actor migration. Non-cooperative
     migration, eg failures, are detected and the non-critical data is
     lazily reconstructed at the cost of a temporary infidelity in the
     accuracy of the simulation.
+ Company
   + The Company provides top-level simulation control.  It hires employees
     into Sites and maintains the "BluePages" that accurately lists all
     non-retired Researchers.
   + The Company code illustrates use of the actor state APIs to eagerly
     save an Actor instance's state on virtually every update to ensure
     correct recovery from failure.

## Running locally

The first time you run locally, you will need to execute `npm install`.

The easiest way to run locally is to invoke the script `./deploy/runServerLocally.sh` in one shell terminal and `./deploy/runClientLocally.sh` in another terminal. This will simulate several work days across three IBM Research
sites in approximately a minute.  Typical output is shown below:
```
$ ./deploy/runClientLocally.sh
2020/04/24 16:51:23 [WARNING] starting...
2020/04/24 16:51:26 [INFO] setup session for topic kar_ykt, generation 2, claims []
2020/04/24 16:51:26 [INFO] increasing partition count for topic kar_ykt to 2
2020/04/24 16:51:26 [INFO] cleanup session for topic kar_ykt, generation 2
2020/04/24 16:51:28 [INFO] setup session for topic kar_ykt, generation 3, claims [1]
2020/04/24 16:51:28 [INFO] KAR_RUNTIME_PORT=30666 KAR_APP_PORT=8080
2020/04/24 16:51:28 [INFO] launching service...
2020/04/24 16:51:28 [INFO] rebalanceReminders: responsibility unchanged (responsible = false)
2020/04/24 16:51:30 [STDOUT] Starting simulation: {"Yorktown":{"workers":20,"thinkms":2000,"steps":10,"days":2},"Cambridge":{"workers":10,"thinkms":1000,"steps":40,"days":1},"Almaden":{"workers":15,"thinkms":500,"steps":10,"days":5}}
2020/04/24 16:51:36 [STDOUT] Num employees is 45
2020/04/24 16:51:41 [STDOUT] Num employees is 45
2020/04/24 16:51:46 [STDOUT] Num employees is 45
2020/04/24 16:51:51 [STDOUT] Num employees is 45
2020/04/24 16:51:56 [STDOUT] Num employees is 45
2020/04/24 16:52:01 [STDOUT] Num employees is 45
2020/04/24 16:52:06 [STDOUT] Num employees is 45
2020/04/24 16:52:11 [STDOUT] Num employees is 45
2020/04/24 16:52:16 [STDOUT] Num employees is 41
2020/04/24 16:52:21 [STDOUT] Num employees is 20
2020/04/24 16:52:26 [STDOUT] Num employees is 14
2020/04/24 16:52:31 [STDOUT] Num employees is 0
2020/04/24 16:52:31 [STDOUT] Valiadating Yorktown
2020/04/24 16:52:31 [STDOUT] Reminder Delays for Yorktown
2020/04/24 16:52:31 [STDOUT] 	<100ms	19
2020/04/24 16:52:31 [STDOUT] 	<200ms	15
2020/04/24 16:52:31 [STDOUT] 	<300ms	21
2020/04/24 16:52:31 [STDOUT] 	<400ms	22
2020/04/24 16:52:31 [STDOUT] 	<500ms	30
2020/04/24 16:52:31 [STDOUT] 	<600ms	55
2020/04/24 16:52:31 [STDOUT] 	<700ms	43
2020/04/24 16:52:31 [STDOUT] 	<800ms	71
2020/04/24 16:52:31 [STDOUT] 	<900ms	53
2020/04/24 16:52:31 [STDOUT] 	<1000ms	35
2020/04/24 16:52:31 [STDOUT] 	<1100ms	30
2020/04/24 16:52:31 [STDOUT] 	<1200ms	6
2020/04/24 16:52:31 [STDOUT] Valiadating Cambridge
2020/04/24 16:52:31 [STDOUT] Reminder Delays for Cambridge
2020/04/24 16:52:31 [STDOUT] 	<100ms	27
2020/04/24 16:52:31 [STDOUT] 	<200ms	29
2020/04/24 16:52:31 [STDOUT] 	<300ms	22
2020/04/24 16:52:31 [STDOUT] 	<400ms	28
2020/04/24 16:52:31 [STDOUT] 	<500ms	46
2020/04/24 16:52:31 [STDOUT] 	<600ms	68
2020/04/24 16:52:31 [STDOUT] 	<700ms	33
2020/04/24 16:52:31 [STDOUT] 	<800ms	53
2020/04/24 16:52:31 [STDOUT] 	<900ms	39
2020/04/24 16:52:31 [STDOUT] 	<1000ms	29
2020/04/24 16:52:31 [STDOUT] 	<1100ms	21
2020/04/24 16:52:31 [STDOUT] 	<1200ms	5
2020/04/24 16:52:31 [STDOUT] Valiadating Almaden
2020/04/24 16:52:31 [STDOUT] Reminder Delays for Almaden
2020/04/24 16:52:31 [STDOUT] 	<100ms	113
2020/04/24 16:52:31 [STDOUT] 	<200ms	82
2020/04/24 16:52:31 [STDOUT] 	<300ms	31
2020/04/24 16:52:31 [STDOUT] 	<400ms	57
2020/04/24 16:52:31 [STDOUT] 	<500ms	64
2020/04/24 16:52:31 [STDOUT] 	<600ms	86
2020/04/24 16:52:31 [STDOUT] 	<700ms	78
2020/04/24 16:52:31 [STDOUT] 	<800ms	70
2020/04/24 16:52:31 [STDOUT] 	<900ms	68
2020/04/24 16:52:31 [STDOUT] 	<1000ms	61
2020/04/24 16:52:31 [STDOUT] 	<1100ms	28
2020/04/24 16:52:31 [STDOUT] 	<1200ms	10
2020/04/24 16:52:31 [STDOUT] 	<1300ms	2
2020/04/24 16:52:31 [STDOUT] SUCCESS
2020/04/24 16:52:31 [STDOUT] Requesting server shutdown
2020/04/24 16:52:31 [STDOUT] Terminating sidecar
2020/04/24 16:52:31 [INFO] Invoking cancel() in response to kill request
2020/04/24 16:52:32 [INFO] cleanup session for topic kar_ykt, generation 3
2020/04/24 16:52:32 [INFO] service exited normally
2020/04/24 16:52:32 [WARNING] exiting...
```

```
$ ./deploy/runServerLocally.sh
Daves-MacBook-Pro:actors-ykt dgrove$ ./deploy/runServerLocally.sh 
2020/04/24 16:51:09 [WARNING] starting...
2020/04/24 16:51:11 [INFO] setup session for topic kar_ykt, generation 1, claims [0]
2020/04/24 16:51:11 [INFO] KAR_RUNTIME_PORT=58417 KAR_APP_PORT=8080
2020/04/24 16:51:11 [INFO] launching service...
2020/04/24 16:51:11 [INFO] rebalanceReminders: change in role: prior = false current = true
2020/04/24 16:51:11 [INFO] rebalanceReminders: found 0 persisted reminders
2020/04/24 16:51:11 [INFO] rebalanceReminders: operation completed
2020/04/24 16:51:26 [INFO] cleanup session for topic kar_ykt, generation 1
2020/04/24 16:51:26 [INFO] setup session for topic kar_ykt, generation 2, claims [0]
2020/04/24 16:51:26 [INFO] increasing partition count for topic kar_ykt to 2
2020/04/24 16:51:26 [INFO] cleanup session for topic kar_ykt, generation 2
2020/04/24 16:51:28 [INFO] setup session for topic kar_ykt, generation 3, claims [0]
2020/04/24 16:51:28 [INFO] rebalanceReminders: responsibility unchanged (responsible = true)
2020/04/24 16:51:30 [STDOUT] 20 hired to perform 10 tasks/day for 2 days at Yorktown
2020/04/24 16:51:30 [STDOUT] 10 hired to perform 40 tasks/day for 1 days at Cambridge
2020/04/24 16:51:31 [STDOUT] 15 hired to perform 10 tasks/day for 5 days at Almaden
2020/04/24 16:51:31 [STDOUT] {
2020/04/24 16:51:31 [STDOUT]   site: 'Yorktown',
2020/04/24 16:51:31 [STDOUT]   siteEmployees: 20,
2020/04/24 16:51:31 [STDOUT]   time: 'Fri Apr 24 2020 16:51:31 GMT-0400 (Eastern Daylight Time)',
2020/04/24 16:51:31 [STDOUT]   delaysBucketMS: 100,
2020/04/24 16:51:31 [STDOUT]   reminderDelays: [],
2020/04/24 16:51:31 [STDOUT]   onboarding: 0,
2020/04/24 16:51:31 [STDOUT]   home: 0,
2020/04/24 16:51:31 [STDOUT]   commuting: 19,
2020/04/24 16:51:31 [STDOUT]   working: 0,
2020/04/24 16:51:31 [STDOUT]   meeting: 0,
2020/04/24 16:51:31 [STDOUT]   coffee: 1,
2020/04/24 16:51:31 [STDOUT]   lunch: 0
2020/04/24 16:51:31 [STDOUT] }
2020/04/24 16:51:32 [STDOUT] {
2020/04/24 16:51:32 [STDOUT]   site: 'Cambridge',
2020/04/24 16:51:32 [STDOUT]   siteEmployees: 10,
2020/04/24 16:51:32 [STDOUT]   time: 'Fri Apr 24 2020 16:51:32 GMT-0400 (Eastern Daylight Time)',
2020/04/24 16:51:32 [STDOUT]   delaysBucketMS: 100,
2020/04/24 16:51:32 [STDOUT]   reminderDelays: [],
2020/04/24 16:51:32 [STDOUT]   onboarding: 0,
2020/04/24 16:51:32 [STDOUT]   home: 0,
2020/04/24 16:51:32 [STDOUT]   commuting: 8,
2020/04/24 16:51:32 [STDOUT]   working: 0,
2020/04/24 16:51:32 [STDOUT]   meeting: 2,
2020/04/24 16:51:32 [STDOUT]   coffee: 0,
2020/04/24 16:51:32 [STDOUT]   lunch: 0
2020/04/24 16:51:32 [STDOUT] }
2020/04/24 16:51:32 [STDOUT] {
2020/04/24 16:51:32 [STDOUT]   site: 'Almaden',
2020/04/24 16:51:32 [STDOUT]   siteEmployees: 15,
2020/04/24 16:51:32 [STDOUT]   time: 'Fri Apr 24 2020 16:51:32 GMT-0400 (Eastern Daylight Time)',
2020/04/24 16:51:32 [STDOUT]   delaysBucketMS: 100,
2020/04/24 16:51:32 [STDOUT]   reminderDelays: [],
2020/04/24 16:51:32 [STDOUT]   onboarding: 0,
2020/04/24 16:51:32 [STDOUT]   home: 0,
2020/04/24 16:51:32 [STDOUT]   commuting: 13,
2020/04/24 16:51:32 [STDOUT]   working: 1,
2020/04/24 16:51:32 [STDOUT]   meeting: 1,
2020/04/24 16:51:32 [STDOUT]   coffee: 0,
2020/04/24 16:51:32 [STDOUT]   lunch: 0
2020/04/24 16:51:32 [STDOUT] }
2020/04/24 16:51:37 [STDOUT] {
2020/04/24 16:51:37 [STDOUT]   site: 'Yorktown',
2020/04/24 16:51:37 [STDOUT]   siteEmployees: 20,
2020/04/24 16:51:37 [STDOUT]   time: 'Fri Apr 24 2020 16:51:37 GMT-0400 (Eastern Daylight Time)',
2020/04/24 16:51:37 [STDOUT]   delaysBucketMS: 100,
2020/04/24 16:51:37 [STDOUT]   reminderDelays: [],
2020/04/24 16:51:37 [STDOUT]   onboarding: 0,
2020/04/24 16:51:37 [STDOUT]   home: 0,
2020/04/24 16:51:37 [STDOUT]   commuting: 0,
2020/04/24 16:51:37 [STDOUT]   working: 12,
2020/04/24 16:51:37 [STDOUT]   meeting: 5,
2020/04/24 16:51:37 [STDOUT]   coffee: 1,
2020/04/24 16:51:37 [STDOUT]   lunch: 2
2020/04/24 16:51:37 [STDOUT] }
2020/04/24 16:51:37 [STDOUT] {
2020/04/24 16:51:37 [STDOUT]   site: 'Cambridge',
2020/04/24 16:51:37 [STDOUT]   siteEmployees: 10,
2020/04/24 16:51:37 [STDOUT]   time: 'Fri Apr 24 2020 16:51:37 GMT-0400 (Eastern Daylight Time)',
2020/04/24 16:51:37 [STDOUT]   delaysBucketMS: 100,
2020/04/24 16:51:37 [STDOUT]   reminderDelays: [],
2020/04/24 16:51:37 [STDOUT]   onboarding: 0,
2020/04/24 16:51:37 [STDOUT]   home: 0,
2020/04/24 16:51:37 [STDOUT]   commuting: 0,
2020/04/24 16:51:37 [STDOUT]   working: 6,
2020/04/24 16:51:37 [STDOUT]   meeting: 2,
2020/04/24 16:51:37 [STDOUT]   coffee: 1,
2020/04/24 16:51:37 [STDOUT]   lunch: 1
2020/04/24 16:51:37 [STDOUT] }
2020/04/24 16:51:38 [STDOUT] {
2020/04/24 16:51:38 [STDOUT]   site: 'Almaden',
2020/04/24 16:51:38 [STDOUT]   siteEmployees: 15,
2020/04/24 16:51:38 [STDOUT]   time: 'Fri Apr 24 2020 16:51:38 GMT-0400 (Eastern Daylight Time)',
2020/04/24 16:51:38 [STDOUT]   delaysBucketMS: 100,
2020/04/24 16:51:38 [STDOUT]   reminderDelays: [],
2020/04/24 16:51:38 [STDOUT]   onboarding: 0,
2020/04/24 16:51:38 [STDOUT]   home: 0,
2020/04/24 16:51:38 [STDOUT]   commuting: 0,
2020/04/24 16:51:38 [STDOUT]   working: 11,
2020/04/24 16:51:38 [STDOUT]   meeting: 3,
2020/04/24 16:51:38 [STDOUT]   coffee: 1,
2020/04/24 16:51:38 [STDOUT]   lunch: 0
2020/04/24 16:51:38 [STDOUT] }
.
.
...3 site report messages every 5 seconds...
.
.
2020/04/24 16:52:31 [STDOUT] Shutting down service
2020/04/24 16:52:31 [INFO] Invoking cancel() in response to kill request
2020/04/24 16:52:32 [INFO] cleanup session for topic kar_ykt, generation 3
2020/04/24 16:52:36 [INFO] service exited normally
2020/04/24 16:52:36 [WARNING] exiting...

```

### Running the site report publication to Slack

The site report can be aggregated and published to Slack. To do so, all site reports
are published to the `siteReport` Kafka topic, an aggregator process will consume
the reports and send update messages on the `outputReport` topic to Slack.

To set up this part of the example a valid kamel installation is required as detailed
in the camel-k example.

To output to Slack export the Slack webhook to the following environment variable:

```
export SLACK_KAR_OUTPUT_WEBHOOK=<webhook-url>
```

To Slack component also needs to connect to KAR's Kafka instance. To do so, export the
following environment variables to contain the cluster IP address of the service:

```
export KAR_KAFKA_CLUSTER_IP=X.X.X.X
```

Create the Kafka topics used by this part fo the example:

```
sh createTopics.sh
```

To start the Slack output process run:
```
./deploy/runOutputToSlack.sh
```

To run the aggregator process:
```
./deploy/runReportAggregator.sh
```

## Running in Kubernetes

There is a Helm chart to deploy the simulation on Kubernetes.  By
default it creates a `Deployment` with two replicas to execute the
simulation.  The client is configured as a Helm test that initiates a
single workday and verifies that the expected work was completed.

```
$ helm install ykt ./deploy/chart
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
