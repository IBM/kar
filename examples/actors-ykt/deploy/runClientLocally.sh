#!/bin/bash

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
CODEDIR="$SCRIPTDIR/.."

KAR_VERBOSE=${KAR_VERBOSE:="info"}
KAR_RECV_PORT=${KAR_REVC_PORT:=30666}

kar  -v $KAR_VERBOSE -recv $KAR_RECV_PORT -app ykt -service client node $CODEDIR/ykt-client.js
