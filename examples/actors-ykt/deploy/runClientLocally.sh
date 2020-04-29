#!/bin/bash

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
CODEDIR="$SCRIPTDIR/.."

KAR_VERBOSE=${KAR_VERBOSE:="info"}
KAR_RUNTIME_PORT=${KAR_RUNTIME_PORT:=30666}

kar  -v $KAR_VERBOSE -runtime_port $KAR_RUNTIME_PORT -partition_zero_ineligible -app ykt -service client node $CODEDIR/ykt-client.js
