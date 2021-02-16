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

# Service timeout scenario

A test case for fault tolerance to demonstrate that
server A making calls to service B via the KAR REST
API will not notice that service B fails and is restarted.



## Building
Build the application components by doing `mvn package`


## Run using KAR

1. Launch the Backend Server
```shell
cd server-back
kar run -app jst -service_timeout 3600s -app_port 9080 -service backend mvn liberty:run
```

2. Launch the Frontend Server
```shell
cd server-back
kar run -app jst -service_timeout 3600s -app_port 9081 -service frontend mvn liberty:run
```

3. Invoke the runTest method on the frontend
```shell
kar rest -app jst -service_timeout 3600s post frontend runTest '{"count":5, "delay":5}'
```

4. Wait for a request to be processed; then kill the backend server in
the middle of processing a request.

5. After some time (but less than the 3600s timeout), restart the
backend server. The inflight request should be re-issued and the
frontend server should resume and successfully complete its
computation.
