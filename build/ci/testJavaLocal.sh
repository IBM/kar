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
cd $ROOTDIR/examples/java/actors
mvn clean install

echo "Creating Java actor server"
cd $ROOTDIR/examples/java/actors/kar-actor-example
mvn liberty:create liberty:install-feature liberty:deploy liberty:package -Dinclude=runnable

echo "Launching Java actor server"
cd $ROOTDIR/examples/java/actors/kar-actor-example/target
kar -v info  -actor_reminder_interval 30s -app actor -service dummy -actors dummy,dummy2 java -jar kar-actor-example.jar &
PID=$!

echo "Waiting 10 seconds for Java actor server to launch"
sleep 10

echo "Sending curl request to Java actor server"
run $PID kar -runtime_port 32123 -app actor curl --fail -H "Content-Type: application/kar+json" -X POST http://localhost:32123/kar/v1/actor/dummy/dummyid/call/canBeInvoked -d '[{ "number": 10}]'

