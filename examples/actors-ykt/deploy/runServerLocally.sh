#!/bin/bash

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
CODEDIR="$SCRIPTDIR/.."

VERBOSE=${VERBOSE:="debug"}

kar -actor_reminder_interval=60s -v $VERBOSE -app ykt -service simulation -actors Site,Floor,Office,Researcher node $CODEDIR/ykt.js
