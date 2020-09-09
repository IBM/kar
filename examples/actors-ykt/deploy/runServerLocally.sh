#!/bin/bash

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
CODEDIR="$SCRIPTDIR/.."

KAR_VERBOSE=${KAR_VERBOSE:="info"}
KAR_ACTOR_REMINDER_INTERVAL=${KAR_ACTOR_REMINDER_INTERVAL:="100ms"}

kar run -v $KAR_VERBOSE -actor_reminder_interval $KAR_ACTOR_REMINDER_INTERVAL -app ykt -service simulation -actors Company,Site,Office,Researcher node $CODEDIR/ykt.js
