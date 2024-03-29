#!/bin/bash

#
# Copyright IBM Corporation 2020,2023
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
CODEDIR="$SCRIPTDIR/.."

KAR_VERBOSE=${KAR_VERBOSE:="info"}
KAR_ACTOR_REMINDER_INTERVAL=${KAR_ACTOR_REMINDER_INTERVAL:="100ms"}
KAR_EXTRA_ARGS=${KAR_EXTRA_ARGS:=""}
KAR_DEBUG=${KAR_DEBUG:=""}

kar run $KAR_DEBUG -v $KAR_VERBOSE -actor_reminder_interval $KAR_ACTOR_REMINDER_INTERVAL -app ykt -actors Company,Site,Office,Researcher $KAR_EXTRA_ARGS node $CODEDIR/ykt.js
