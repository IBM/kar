#!/bin/bash

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

. $ROOTDIR/scripts/kar-kind-env.sh

echo "Building Java KAR SDK"
cd $ROOTDIR/sdk-java
mvn clean install

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


