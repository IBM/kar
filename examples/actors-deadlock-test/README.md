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

This example generates a deadlock, which can be inspected using
`kar-debugger`.

To run the example locally, first do an `npm install`.
Then in one window start up the server code:
```shell
kar run -app dp -actors ActorTypeA,ActorTypeB,Tester node philosophers.js
```

In a second window, start the debugger server (see core/cmd/kar-debugger/README.md).

Then, in a third window, use the `kar` cli to start the deadlock testing client:
```shell
kar invoke -app dp Tester TesterX startTest
```

And in a fourth window, once the deadlock occurs, view it using the
debugger command:
```shell
kar-debugger vd
```


