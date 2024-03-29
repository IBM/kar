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

## KAR Unit Tests

### Running locally

```sh
npm install

# run server
kar run -app myApp -service myService -actors Foo node server.js &

# run a trivial client
kar run -app myApp node client.js

# run the test suite client
kar run -app myApp node test-harness.js
```

### Running on Kubernetes

```shell
$ helm install ut ./deploy/chart --set image=example-js:dev
NAME: ut
LAST DEPLOYED: Fri Apr  3 16:34:15 2020
NAMESPACE: default
STATUS: deployed
REVISION: 1
$ kubectl get pods
NAME                         READY   STATUS    RESTARTS   AGE
ut-server-6fb75d6b55-nrnbn   2/2     Running   0          5s
ut-server-6fb75d6b55-qznl8   2/2     Running   0          5s
$ helm test ut
Pod ut-client pending
Pod ut-client pending
Pod ut-client running
Pod ut-client succeeded
NAME: ut
LAST DEPLOYED: Fri Apr  3 16:34:15 2020
NAMESPACE: default
STATUS: deployed
REVISION: 1
TEST SUITE:     ut-client
Last Started:   Fri Apr  3 16:34:25 2020
Last Completed: Fri Apr  3 16:34:34 2020
Phase:          Succeeded
$ helm delete ut
release "ut" uninstalled
```
