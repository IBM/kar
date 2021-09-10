#!/bin/bash

#
# Copyright IBM Corporation 2020,2021
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

# Script to run a KAR component on Code Engine

set -e

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/.."

app=""
image=""
actors=""
scale=1
service=""
verbose="error"
port="8080"
runargs=""

help=""
args=""
parse=true

while [ -n "$1" ]; do
    if [ -z "$parse" ]; then
        args="$args '$1'"
        shift
        continue
    fi
    case "$1" in
        -h|-help|--help) help="1"; break;;
        -app)
            shift;
            app="$1"
            ;;
        -actors)
            shift;
            actors="$1"
            ;;
        -image)
            shift;
            image="$1"
            ;;
        -port)
            shift;
            port="$1"
            ;;
        -env)
            shift;
            runargs="$runargs --env $1"
            ;;
        -service)
            shift;
            service="$1"
            ;;
        -scale)
            shift;
            scale=$1
            ;;
        -v)
            shift;
            verbose="$1"
            ;;
        --) parse=;;
        *) args="$args '$1'";;
    esac
    shift
done

if [ -n "$help" ]; then
    cat << EOF
Usage: kar-ce-run.sh [options]
where [options] includes:
    -app <appname>            invoke kar with -app <appname>   (required)
    -image <image>            container image to run           (required)
    -actors <actors>          invoke kar with -actors <actors>
    -service <service>        invoke kar with -service <service>
    -port <port>              invoke kar with -app_port <port> (default 8080)
    -env KEY=VALUE            add the binding KEY=VALUE to the container's environment
    -scale <N>                run N replicas of this component (default 1)
    -v <level>                invoke kar with -v <level>       (default error)
EOF
    exit 0
fi

# Check that required arguments were given
if [ "$app" == "" ]; then
    echo "-app <appname> is a required argument"
    exit 1
fi
if [ "$image" == "" ]; then
    echo "-image <containerimage> is a required argument"
    exit 1
fi

karargs="-app $app -v $verbose -app_port $port"
if [ "$service" != "" ]; then
    karargs="$karargs -service $service"
    runargs="$runargs --label kar.ibm.com/service=$service"
fi
if [ "$actors" != "" ]; then
    karargs="$karargs -actors $actors"
fi

runargs="$runargs --label kar.ibm.com/app=$app --network kar-bus --detach"
runargs="$runargs --env KAFKA_BROKERS=kafka:9092 --env KAFKA_VERSION=2.6.0"
runargs="$runargs --env REDIS_HOST=redis --env REDIS_PORT=6379 --env REDIS_USER=karmesh -env REDIS_PASSWORD=act0rstate"
runargs="$runargs --env KAR_APP=$app --env KAR_SIDECAR_IN_CONTAINER=true --env KAR_APP_PORT=$port"
runargs="$runargs --env KAR_EXTRA_ARGS=\"$karargs\""

echo docker run $runargs $image

for (( i = 0 ;  i < $scale ; i++)); do
    eval docker run $runargs $image
done
