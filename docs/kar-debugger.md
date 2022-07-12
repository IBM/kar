<!--
# Copyright IBM Corporation 2022
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

This document describes support for using the `kar-debugger` program in
order to debug KAR applications.

## Componenents

In order to use the debugger, two components need to be run:

1. The "debugger server".
2. The "debugger client".

The responsibility of the debugger server is to connect to a KAR sidecar,
in order to ferry commands between the debugger client and the sidecar.

The debugger client is what you run every time you send a debugger command.
It talks to the debugger server.

## Setting up the debugger server

The debugger server is run using the following command:

```shell
kar-debugger server sidecarHostname sidecarPort
```

where `sidecarHostname` is the hostname of the KAR sidecar to which you want
to connect, and `sidecarPort` is the port. You can find the hostnames and ports
of sidecars to connect to by using the command

```shell
kar get -app appName -s sidecars
```

which will list all sidecars along with their hostnames and ports. This is useful
for local development, but can be inconvenient when the application is running
in the cloud. For that case, you are advised to run the debugger server as a KAR
process.

Note that once the debugger server connects, it will act as a monitor, printing
out information whenever breakpoints are hit, or actors are paused, etc.

Additionally, you can change the port on which the debugger server listens.
By default, it listens on port 5364. You can change this port, however, using the
`-serverPort` option.

### Running the debugger server as a KAR process

It is possible to lauch the debugger server as a KAR process running in its own
sidecar. This is useful in cases where the KAR sidecars are not exposed to the
outside world, but a single port can be. You can run the debugger server in this
way as follows:

```shell
kar run -app appName kar-debugger server
```

Note that no sidecarHostname or sidecarPort arguments are necessary; the
debugger server will automatically connect to the KAR sidecar that launched it.

When the server is run within a cloud cluster in this way, it might be difficult
to access its STDOUT. As such, you will not be able to use the monitoring
functionality of the debugger. This will be fixed in a future release.

### Debug behavior on server (dis)connection
Whenever a debugger server disconnects from the sidecar, by default, all actors
are unpaused and all breakpoints are unset. If this behavior is undesirable, you
may disable it by launching your KAR application with the `-debug` flag.

#### Note on indirect pauses and debugger server connection (feel free to skip)

If you connect the debugger server to a running application, after
it has already begun processing things, then it is possible that querying the list
of paused actors will not return all actors that are indirectly paused.

This is because indirect pause detection relies on the ability to track which
actors are currently calling which methods. But by default, when no debugger is
connected, this information is not recorded, due to performance concerns. Thus, if
the debugger server connects after actors have already started calling other actors,
then if a breakpoint is hit, the debugger will not be aware that the actors that
were in progress at the time of connection might be waiting on a paused actor.

## Running debugger client commands

All interaction with the debugger is done via client commands. They look like
this:

```shell
kar-debugger b TestActor testMethod -type node
```

This command sets a node-level breakpoint which is triggered whenever the
method `testMethod` of an actor of type `TestActor` is called.

In order to run comands like this, you must be able to connect to the
debugger server. This means that you must know the hostname and port to which 
you want to connect. Once this is known, you can either:

1. pass the host and port to every debugger command, using the `-host` and
`-port` options; or
2. set the `KAR_DEBUGGER_HOST` and `KAR_DEBUGGER_PORT` environment variables.

If none of these options are given, the debugger client by default attempts to
connect to `localhost:5364`. This makes local debugging easier.
