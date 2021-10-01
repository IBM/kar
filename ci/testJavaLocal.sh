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
kar run -v info -app java-hello -service greeter $KAR_EXTRA_ARGS mvn liberty:run &
PID=$!

# Sleep 10 seconds to given liberty time to come up
sleep 10

echo "Run the Hello Client to check invoking a route on the Hello Server"
cd $ROOTDIR/examples/service-hello-java/client
run $PID kar run -app java-hello $KAR_EXTRA_ARGS java -jar target/kar-hello-client-jar-with-dependencies.jar


#################

echo "Building Java Dining Philsopophers against released Java-SDK"
cd $ROOTDIR/examples/actors-dp-java
mvn clean package

echo "Launching Java DP Server"
kar run -v info -app dp -actors Cafe,Fork,Philosopher,Table $KAR_EXTRA_ARGS mvn liberty:run &
PID=$!

# Sleep 10 seconds to give liberty time to come up
sleep 10

echo "Building and launching test harness"
cd $ROOTDIR/examples/actors-dp-js
npm install --prod
run $PID kar run -app dp $KAR_EXTRA_ARGS node tester.js

#################

echo "Building Java Reactive Dining Philsopophers against released Java-SDK"
cd $ROOTDIR/examples/actors-dp-java-reactive
mvn clean package

echo "Launching Java DPR Server"
kar run -v info -app dpr -actors Cafe,Fork,Philosopher,Table $KAR_EXTRA_ARGS java -jar target/quarkus-app/quarkus-run.jar &
PID=$!

# Sleep 5 seconds to give quarkus time to come up
sleep 5

echo "Building and launching test harness"
cd $ROOTDIR/examples/actors-dp-js
npm install --prod
run $PID kar run -app dpr $KAR_EXTRA_ARGS node tester.js

#################

echo "Building Java KAR SDK locally"
cd $ROOTDIR/sdk-java
mvn clean
mvn versions:set -DnewVersion=99.99.99-SNAPSHOT
mvn install

#################

echo "Building Java Dining Philsopophers against local Java KAR SDK"
cd $ROOTDIR/examples/actors-dp-java
mvn -Dversion.kar-java-sdk=99.99.99-SNAPSHOT clean package

echo "Launching Java DP Server"
kar run -v info -app dp-local -actors Cafe,Fork,Philosopher,Table $KAR_EXTRA_ARGS mvn liberty:run -Dversion.kar-java-sdk=99.99.99-SNAPSHOT &
PID=$!

# Sleep 10 seconds to give liberty time to come up
sleep 10

echo "Building and launching test harness"
cd $ROOTDIR/examples/actors-dp-js
npm install --prod
run $PID kar run -app dp-local $KAR_EXTRA_ARGS node tester.js

#################

echo "Building Java Reactive Dining Philsopophers against local Java KAR SDK"
cd $ROOTDIR/examples/actors-dp-java-reactive
mvn -Dversion.kar-java-sdk=99.99.99-SNAPSHOT clean package

echo "Launching Java DPR Server"
kar run -v info -app dpr-local -actors Cafe,Fork,Philosopher,Table $KAR_EXTRA_ARGS java -jar target/quarkus-app/quarkus-run.jar &
PID=$!

# Sleep 5 seconds to give Quarkus time to come up
sleep 5

echo "Building and launching test harness"
cd $ROOTDIR/examples/actors-dp-js
npm install --prod
run $PID kar run -app dpr-local $KAR_EXTRA_ARGS node tester.js
