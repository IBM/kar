#!/bin/bash

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
CODEDIR="$SCRIPTDIR/.."

KAR_VERBOSE=${KAR_VERBOSE:="info"}

kar  -v $KAR_VERBOSE -recv 30666 -app ykt -service client node $CODEDIR/ykt-client.js
