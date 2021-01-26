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

# Notes for KAR Developers

This file collects various hints and tips that are useful to
people who are developing the KAR runtime system, but are not
relevant to using KAR to build applications.

### Local Development - JavaScript SDK - Yalc

We use [yalc](https://www.npmjs.com/package/yalc) to keep the example packages
and the JavaScript SDK package in sync. When making and testing local changes to
the JavaScript SDK these changes need to be propagated to the examples projects
using `yalc`. First install `yalc`:
```shell
$ npm i -g yalc
```
Then configure `yalc` for `KAR`:
```shell
./scripts/setup-yalc.sh
```
Finally, whenever a change is made to the JavaScript SDK run:
```shell
cd sdk-js
yalc push
```

### Local Development - Running test cases

The scripts in the `ci` directory are a good way
execute test cases during development.

### Swagger API documentation

We generate Swagger documentating the KAR REST APIs
from comments/markup in the go code in core/internal/runtime.

The generated files are committed to git in docs/api to
make it possible to serve them from https://ibm.github.io/kar/.

To regenerate the swagger, do
```shell
make swagger-gen
```

To browse the API locally, do
```shell
make swagger-serve
```
