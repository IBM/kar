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

# Script to run a KAR component on IBM Code Engine

set -e

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/.."

app=""
image=""
actors=""
scale="1"
service=""
verbose="error"
port="8080"
ceargs=""
cluster_local="--cluster-local"
nowait=""

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
            ceargs="$ceargs --env $1"
            ;;
        -externalize)
            cluster_local=""
            ;;
        -nowait)
            nowait="--no-wait"
            ;;
        -name)
            shift;
            name="$1"
            ;;
        -registry-secret)
            shift;
            ceargs="$ceargs --registry-secret $1"
            ;;
        -service)
            shift;
            service="$1"
            ;;
        -scale)
            shift;
            scale="$1"
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
    -name <componentname>     name for this app component      (required)
    -actors <actors>          invoke kar with -actors <actors>
    -service <service>        invoke kar with -service <service>
    -port <port>              invoke kar with -app_port <port> (default 8080)
    -externalize              make app port accessible via public URL (default cluster_local)
    -registry-secret <sec>    name of Code Engine registry secret to use
    -env KEY=VALUE            add the binding KEY=VALUE to the container's environment
    -scale <N>                run N replicas of this component (default 1)
    -v <level>                invoke kar with -v <level>       (default error)
    -nowait                   create the app asynchronously    (default wait up to 300 seconds)
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
if [ "$name" == "" ]; then
    echo "-name <componentname> is a required argument"
    exit 1
fi

# Build kar command line options
karargs="-app $app -v $verbose -app_port $port"
if [ "$service" != "" ]; then
    karargs="$karargs -service $service"
fi
if [ "$actors" != "" ]; then
    karargs="$karargs -actors $actors"
fi

ceargs="$ceargs --image $image --name $name --min-scale $scale --max-scale $scale --cpu 1 --port http1:$port"
ceargs="$ceargs --env-from-secret kar.ibm.com.runtime-config $cluster_local $nowait"
ceargs="$ceargs --env KAR_APP=$app --env KAR_SIDECAR_IN_CONTAINER=true --env KAR_APP_PORT=$port"
ceargs="$ceargs --env KAR_EXTRA_ARGS=\"$karargs\""

echo ibmcloud ce app create $ceargs
eval ibmcloud ce app create $ceargs
