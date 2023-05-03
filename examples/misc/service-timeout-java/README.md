<!--
# Copyright IBM Corporation 2020,2023
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

A skeleton of the component interactions found in the reefer application.
A front end component invokes a KAR service in the middle server,
which in turn invokes two actor calls on a backend server.
Each call can be configured with think times for the middle and
backend.  By setting these values and/or killing and restarting
processes we can simulate a variety of failure and timeout
scenarios in a simple environment.


## Building
Build the application components by doing `mvn package`


## Run using KAR

1. Launch the Backend Server
```shell
cd server-back
kar run -app jst -app_port 9080 -actors SlowAdder mvn liberty:run
```

2. Launch the Middle Server
```shell
cd server-middle
kar run -app jst -app_port 9081 -service middle mvn liberty:run
```

3. Launch the Frontend Server
```shell
cd server-front
kar run -app jst -app_port 9082 -service frontend mvn liberty:run
```

4. Invoke the runTest method on the frontend
```shell
kar rest -app jst post frontend runTest '{"count":5, "delay":5}'
```

5. Wait for a request to be processed; then kill the middle and/or
backend server in the middle of processing a request.

6. After an arbitray amount of time (seconds. minutes or hours)
restart the killed server(s). The inflight request should be re-issued
and the frontend server should resume and successfully complete its
computation.
