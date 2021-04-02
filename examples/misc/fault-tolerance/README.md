<!--
# Copyright IBM Corporation 2020,2021
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

# Fault-Tolerance in Action

An example program to see KAR's fault-tolerance in action.

In one window run:
```shell
kar run -app echo -service echo node server.js
```

In a second window run:
```shell
kar run -app echo node client.js
```

In the first window, kill the server while the second request is being
processed. Then restart the server with the same command line as before. The new
server instance should restart the interrupted request and respond to the
client. The client should be unaffected.
