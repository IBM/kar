#!/bin/bash

#
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
#

set -meu

# run background_services_gpid test_command
run () {
    PID=$1
    shift
    CODE=0
    "$@" || CODE=$?
    kill -- -$PID || true
    sleep 1
    return $CODE
}

echo "Executing Java Hello Service test"

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/.."

. $ROOTDIR/scripts/kar-env-local.sh

echo "Building Java Hello Service"
cd $ROOTDIR/examples/service-hello-java
mvn clean package

echo "Launching Java Hello Server"
cd $ROOTDIR/examples/service-hello-java/server
kar run -v info -app java-hello -service greeter mvn liberty:run &
PID=$!

# Sleep 10 seconds to given liberty time to come up
sleep 10

echo "Run the Hello Client to check invoking a route on the Hello Server"
cd $ROOTDIR/examples/service-hello-java/client
run $PID kar run -app java-hello java -jar target/kar-hello-client-jar-with-dependencies.jar


#################

echo "Building Java Dining Philsopophers against released Java-SDK"
cd $ROOTDIR/examples/actors-dp-java
mvn clean package

echo "Launching Java DP Server"
kar run -v info -app dp -actors Cafe,Fork,Philosopher,Table mvn liberty:run &
PID=$!

# Sleep 10 seconds to given liberty time to come up
sleep 10

echo "Building and launching test harness"
cd $ROOTDIR/examples/actors-dp-js
npm install --prod
run $PID kar run -app dp node tester.js

#################

echo "Building Java KAR SDK locally"
cd $ROOTDIR/sdk-java
mvn clean install

echo "Building Java Dining Philsopophers against local Java KAR SDK"
cd $ROOTDIR/examples/actors-dp-java
mvn -Dversion.kar-java-sdk=0.6.1-SNAPSHOT clean package

echo "Launching Java DP Server"
kar run -v info -app dp-local -actors Cafe,Fork,Philosopher,Table mvn liberty:run &
PID=$!

# Sleep 10 seconds to given liberty time to come up
sleep 10

echo "Building and launching test harness"
cd $ROOTDIR/examples/actors-dp-js
npm install --prod
run $PID kar run -app dp-local node tester.js
