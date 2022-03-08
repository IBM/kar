<!--
# Copyright IBM Corporation 2020,2022
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

# Python Hello World Example with HTTP2

This examples shows how a Python frontend can be used in conjunction with KAR.

The example uses several technologies that enable asynchronous HTTP2 communication in Python:
    - HTTPX: to perform client-side HTTP2 requests.
    - asyncio: Python package used to control the asynchronous behavior on the client and server sides.
    - Hypercorn: ASGI web server that supports HTTP2.
    - FastAPI: high-performance web framework for building Python APIs.

These technologies are used in conjunction with KAR. To enable the use of HTTP2 communication between the KAR sidecard and the client or server processes, the `-h2c` flag needs to be passed as a KAR option.

In two separate terminals start the server and client processes as follows:

The server side:
```
kar run -h2c -app hello-async -service async-server python server.py
```

The client side:
```
kar run -h2c -app hello-async python client.py
```

The following output can be observed on the client side:
```
2022/02/28 12:11:54.767078 [STDOUT] HTTP/2
2022/02/28 12:11:54.767100 [STDOUT] "Hello John Doe"
2022/02/28 12:11:54.767104 [STDOUT] HTTP/2
2022/02/28 12:11:54.767113 [STDOUT] Hello John Doe
2022/02/28 12:11:54.767117 [STDOUT] HTTP/2
2022/02/28 12:11:54.767119 [STDOUT] I am healthy
```

On the server side several messages displaying the HTTP version will appear confirming the use of HTTP2:
```
2022/02/28 13:08:57.910927 [STDOUT] HTTP version: 2
...
```

The example includes a client that times the request duration in various settings:
```
kar run -h2c -app hello-async python timed-client.py
```
