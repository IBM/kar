#!/bin/bash

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
CODEDIR="$SCRIPTDIR/.."

VERBOSE=${VERBOSE:="debug"}

VERBOSE=1 kar -actor_reminder_interval=5s -v $VERBOSE -app ykt -service simulation -actors Site,Floor,Office,Researcher node $CODEDIR/ykt.js
