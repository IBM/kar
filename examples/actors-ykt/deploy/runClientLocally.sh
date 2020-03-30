#!/bin/bash

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
CODEDIR="$SCRIPTDIR/.."

VERBOSE=${VERBOSE:="debug"}

kar  -v $VERBOSE -recv 30666 -app ykt -service client node $CODEDIR/ykt-client.js

sleep 600

