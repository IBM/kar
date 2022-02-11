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

This example uses KAR's Actor Programming Model to implement a simple use for actors in using the KAR Python SDK.

Launch the server:

```
kar run -app hello-actor -actors FamousActor python3 server.py
```

In another terminal launch the client code:

```
kar run -app hello-actor python3 client.py
```
