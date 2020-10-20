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

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/.."

. $ROOTDIR/scripts/kar-env-local.sh

# Run unit-tests/test-harness.js locally
echo "*** Testing examples/unit-tests ***"

cd $ROOTDIR/examples/unit-tests
npm install --prod

kar run -app myApp -service myService -actors Foo node server.js &
run $! kar run -app myApp node test-harness.js

# Run actors-dp-java/tester.js locally
echo "*** Testing examples/actors-dp-js ***"

cd $ROOTDIR/examples/actors-dp-js
npm install --prod

kar run -app dp -actors Cafe,Table,Fork,Philosopher node philosophers.js &
run $! kar run -app dp node tester.js

# Run actors-ykt locally
echo "*** Testing examples/actors-ykt ***"

cd $ROOTDIR/examples/actors-ykt
npm install --prod

./deploy/runServerLocally.sh &
run $! ./deploy/runClientLocally.sh
