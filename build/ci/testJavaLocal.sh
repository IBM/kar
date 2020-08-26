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

echo "Executing Java Actor test"

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/../.."

. $ROOTDIR/scripts/kar-kind-env.sh

echo "Building Java KAR SDK"
cd $ROOTDIR/sdk/java/kar-java
mvn clean install

echo "Building Java Actors sample application"
cd $ROOTDIR/examples/java/actors
mvn clean install

echo "Creating Java actor server"
cd $ROOTDIR/examples/java/actors/kar-actor-example
mvn liberty:create liberty:install-feature liberty:deploy liberty:package -Dinclude=runnable

echo "Launching Java actor server"
cd $ROOTDIR/examples/java/actors/kar-actor-example/target
kar -v info -app example -actors sample,calculator java -jar kar-actor-example.jar &
PID=$!

echo "Waiting 10 seconds for Java actor server to launch"
sleep 10

echo "Invoking actor method on Java actor server"
run $PID kar -runtime_port 32123 -app example curl --fail -H "Content-Type: application/kar+json" -X POST http://localhost:32123/kar/v1/actor/sample/abc/call/canBeInvoked -d '[{ "number": 10}]'

