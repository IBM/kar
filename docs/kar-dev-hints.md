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

# Notes for KAR Developers

This file collects various hints and tips that are useful to
people who are developing the KAR runtime system, but are not
relevant to using KAR to build applications.

## Building KAR from source

Download a source tarball or clone this repository. To build the cli run:
```shell
make cli
```
To build and push the docker images to the local registry run:
```shell
make docker
```

## Developing the JavaScript SDK

The JavaScript examples included is this repository are configured to install
the latest release of the `kar-sdk` NPM package.

In order to run these examples against a local copy of the JavaScript SDK code,
it is necessary to alter the `kar-sdk` configuration specified in the
`package.json` files for the examples. We recommend using
[yalc](https://www.npmjs.com/package/yalc) to manage this process. First install
`yalc`:
```shell
$ npm i -g yalc
```
Then configure `yalc` for KAR and the examples projets to use `yalc`:
```shell
./scripts/setup-yalc.sh
```
Finally, whenever a change is made to the JavaScript SDK run:
```shell
cd sdk-js
yalc push
```

## Developing the Java SDK

The Java  examples included is this repository are configured to
depend on the latest release of the `com.ibm.research.kar` maven packages.

To run these examples against a local copy of the Java SDK code,
it is necessary to first build and install a `SNAPSHOT` version of the
Java SDK maven artifacts and then override the build dependency to use them.

First build and install your local Java SDK SNAPSHOT version with
```shell
make installJavaSDK
```

Then execute the maven command to build the example, with
the command line override of `-Dversion.kar-java-sdk=x.y.z-SNAPSHOT`
where `x.y.z-SNAPSHOT` corresponds to the `version` defined in
[sdk-java/pom.xml](../sdk-java/pom.xml).

For example, to run the actors-dp-java server using a locally built
Java SDK, first build it with the version override:
```shell
mvn package -Dversion.kar-java-sdk=x.y.z-SNAPSHOT
```
Then run it normally:
```
kar run -app dp -actors Cafe,Fork,Philosopher,Table mvn liberty:run
```

## Running test cases

The scripts in the `ci` directory are a good way
execute test cases during development.

## Swagger API documentation

We generate Swagger documenting the KAR REST APIs
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
