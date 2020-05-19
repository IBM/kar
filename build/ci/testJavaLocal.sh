#!/bin/bash

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/../.."

. $ROOTDIR/scripts/kar-kind-env.sh

echo "Executing Java Actor test"
cd $ROOTDIR/examples/java/actors

mvn clean install

cd $ROOTDIR/examples/java/actors/actor-server

echo "Launching Java actor server"
kar -v info  -actor_reminder_interval 30s -app actor -service dummy -actors dummy,dummy2 mvn liberty:run &

echo "Waiting 3 minute for server to launch"
sleep 180

echo "Sending curl request to Java actor server"
kar -runtime_port 32123 -app actor curl -H "Content-Type: application/json" -X POST http://localhost:32123/kar/v1/actor/dummy/dummyid/call/canBeInvoked -d '{ "number": 10}'

echo "Stopping Java actor server"
mvn liberty:stop