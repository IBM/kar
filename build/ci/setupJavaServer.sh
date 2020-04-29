#!/bin/bash

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/../.."

echo "Starting Java test server"
cd $ROOTDIR/examples/java/incr/server
mvn liberty:run
sleep 5