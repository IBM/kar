#!/bin/bash

set -ex

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/../.."

# Run loop.js locally
cd $ROOTDIR/examples/incr

npm install

kar -app myApp -service myService -kafka_brokers localhost:31093 -redis_host localhost -redis_port 31379 -redis_password passw0rd node server.js &

sleep 1

kar -app myApp -service myService -kafka_brokers localhost:31093 -redis_host localhost -redis_port 31379 -redis_password passw0rd node loop.js
