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
ROOTDIR="$SCRIPTDIR/../.."

. $ROOTDIR/scripts/kar-kind-env.sh

# Run unit-tests/test-harness.js locally
echo "*** Executing unit-tests/test-harness.js ***"

cd $ROOTDIR/examples/unit-tests
npm install --prod

kar -app myApp -service myService -actors Foo node server.js &
run $! kar -app myApp node test-harness.js

# Run actors-ykt locally
echo "*** Executing actors-ykt/ykt-client.js ***"

cd $ROOTDIR/examples/actors-ykt
npm install --prod

./deploy/runServerLocally.sh &
run $! ./deploy/runClientLocally.sh
