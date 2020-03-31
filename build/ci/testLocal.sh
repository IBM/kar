#!/bin/bash

set -eu

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/../.."

. $ROOTDIR/scripts/kar-kind-env.sh

# Run incr/test-harness.js locally
echo "Executing incr/test-harness.js"

cd $ROOTDIR/examples/incr
npm install

kar -app myApp -service myService -actors Foo node server.js &
sleep 1

kar -app myApp -service myService node test-harness.js
