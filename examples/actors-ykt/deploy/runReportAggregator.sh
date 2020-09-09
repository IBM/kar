#!/bin/bash

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
CODEDIR="$SCRIPTDIR/.."

KAR_VERBOSE=${KAR_VERBOSE:="info"}
KAR_RUNTIME_PORT=${KAR_RUNTIME_PORT:=30667}

kar run -v $KAR_VERBOSE -app_port 8081 -runtime_port $KAR_RUNTIME_PORT -actors SiteReportManager -app ykt node $CODEDIR/ykt-aggregator.js
