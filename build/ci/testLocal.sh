#!/bin/bash

set -eu

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/../.."

. $ROOTDIR/scripts/kar-kind-env.sh

# Run loop.js locally
cd $ROOTDIR/examples/incr

npm install

kar -app myApp -service myService node server.js &

sleep 1

kar -app myApp -service myService -actors Foo node test-harness.js
