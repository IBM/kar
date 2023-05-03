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

# Hello World Example

This example demonstrates how to use KAR's REST API directly. It consists of an
[HTTP server](server.py) implemented using `Flask` and an [HTTP
client](client.py) implemented using Python `requests`.

## Run the Server without KAR and interact via curl:

In one window:
```shell
(%) KAR_APP_PORT=8080 python server.py
```

In a second window, invoke routes using curl
```shell
(%) curl -s -X POST -H "Content-Type: text/plain" http://localhost:8080/helloText -d 'Gandalf the Grey'
Hello Gandalf the Grey
```
```shell
(%) curl -s -X POST -H "Content-Type: application/json" http://localhost:8080/helloJson -d '{"name": "Alan Turing"}'
{"greetings":"Hello Alan Turing"}
```

## Run using KAR

In one window:
```shell
(%) kar run -app hello-python -service greeter python server.py
```

In a second window:
```shell
(%) kar run -app hello-python python client.py
```

You should see output like shown below in both windows:
```
2020/04/02 17:41:23 [STDOUT] Hello John Doe!
2020/04/02 17:41:23 [STDOUT] Hello John Doe!
2020/04/02 17:41:23 [STDOUT] I am healthy
```
The client process will exit, while the server remains running. You
can send another request, or exit the server with a Control-C.

You can use the `kar` cli to invoke a route directly (the content type for request bodies defaults to application/json).
```shell
(%) kar rest -app hello-python post greeter helloJson '{"name": "Alan Turing"}'
2020/10/06 10:04:27.014025 [STDOUT] {"greetings":"Hello Alan Turing!"}
```

Or invoke the `text/plain` route with an explicit content type:
```shell
(%) kar rest -app hello-python -content_type text/plain post greeter helloText 'Gandalf the Grey'
2020/10/06 09:48:29.644326 [STDOUT] Hello Gandalf the Grey
```

If the service endpoint being invoked requires more sophisticated
headers or other features not supported by the `kar rest` command, it
is still possible to use curl. However, the curl command is now using
KAR's REST API to make the service call via a `kar` sidecar.

```shell
(%) kar run -runtime_port 32123 -app hello-python curl -s -X POST -H "Content-Type: text/plain" http://localhost:32123/kar/v1/service/greeter/call/helloText -d 'Gandalf the Grey'
2020/10/06 09:49:45.300122 [STDOUT] Hello Gandalf the Grey
```
