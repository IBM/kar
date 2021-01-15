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

# Script to automate installation of KAR runtime into a Kubernetes cluster

set -e

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/.."

help=""
args=""
parse=true
icr="enabled"
KAR_API_KEY="none"
kartag="latest"
helmargs=""
injectorOnly=""
while [ -n "$1" ]; do
    if [ -z "$parse" ]; then
        args="$args '$1'"
        shift
        continue
    fi
    case "$1" in
        -h|-help|--help) help="1"; break;;
        -d|-dev|--dev)
            helmargs="$helmargs --set-string kar.injector.imageName=localhost:5000/kar-injector --set-string kar.injector.sidecarImageName=localhost:5000/kar"
            icr="disabled"
            ;;
        -m|-managed|--managed)
            shift;
            serviceKey="$1"
            . $SCRIPTDIR/kar-env-ibmcloud.sh $serviceKey
            helmargs="$helmargs --set kafka.internal=false --set redis.internal=false"
            helmargs="$helmargs --set-string kafka.externalConfig.enabletls=true"
            helmargs="$helmargs --set-string kafka.externalConfig.version=2.3.0"
            helmargs="$helmargs --set-string kafka.externalConfig.brokers=$KAFKA_BROKERS"
            helmargs="$helmargs --set-string kafka.externalConfig.password=$KAFKA_PASSWORD"
            helmargs="$helmargs --set-string redis.externalConfig.enabletls=true"
            helmargs="$helmargs --set-string redis.externalConfig.host=$REDIS_HOST"
            helmargs="$helmargs --set-string redis.externalConfig.port=$REDIS_PORT"
            helmargs="$helmargs --set-string redis.externalConfig.password=$REDIS_PASSWORD"
            injectorOnly="true"
            ;;
        -c|-crkey|--crkey)
            shift;
            KAR_API_KEY=$(cat $1 | jq -r .apikey)
            ;;
        -f|-file|--file) shift; helmargs="$helmargs -f $1";;
        -s|-set|--set) shift; helmargs="$helmargs --set $1";;
        -ss|-set-string|--set-string) shift; helmargs="$helmargs --set-string $1";;
        -r|-release|--release) shift; kartag="$1";;
        --) parse=;;
        *) args="$args '$1'";;
    esac
    shift
done

if [ -n "$help" ]; then
    cat << EOF
Usage: kar-deploy.sh [options]
where [options] includes:
    -f -file <config.yaml>    pass `-f <config.yaml>` to `helm install kar`
    -m -managed <service-key> use managed EventStreams and Redis accessed via service-key
    -c -crkey <apikey.json>   apikey.json to read images from us.icr.io/research/kar-dev
    -s -set key=value         pass `--set key=value to `helm install kar`
    -d -dev                   disable configuring access to the IBM Cloud Container Registry
    -r -release <version>     deploy a specific version of kar
EOF
    exit 0
fi

helmargs="--set-string kar.injector.imageTag=$kartag --set-string kar.injector.sidecarImageTag=$kartag $helmargs"

cd $ROOTDIR

kubectl create namespace kar-system 2>/dev/null || true

if [ "$icr" == "enabled" ]; then
    if [ "$KAR_API_KEY" == "none" ]; then
        if ! ibmcloud cr images --restrict research/kar-dev/kar  | grep -q latest; then
            echo "No images found in research/kar-dev namespace"
            echo "Either 'ibmcloud login' to the RIS account or run in -dev mode"
            exit 1
        fi

        if ibmcloud iam service-id kar-cr-reader-id 2>/dev/null 1>/dev/null; then
            echo "Using existing service account kar-cr-reader-id"
        else
            echo "Creating service account kar-cr-reader-id"
            ibmcloud iam service-id-create kar-cr-reader-id --description "Service ID for IBM Cloud Container Registry to enable Reader access to the research/kar-dev namespace" 1>/dev/null
        fi

        KAR_API_KEY=$(ibmcloud iam service-api-key-create kar-cr-reader-key kar-cr-reader-id --description "API key for kar-cr-reader-id to enable Reader access to the research/kar-dev namespace" 2>/dev/null | grep 'API Key' | awk '/ /{print $3}')

        echo "Successfully created API Key=$KAR_API_KEY"
    fi

    kubectl --namespace kar-system create secret docker-registry kar.ibm.com.image-pull --docker-server=us.icr.io --docker-username=iamapikey --docker-email=kar@ibm.com --docker-password=$KAR_API_KEY
fi

helm install kar scripts/helm/kar -n kar-system $helmargs


waitForPod() {
    while true; do
        POD_STATUS=$(kubectl -n kar-system get pods -l name="$1" -o wide | grep "$1" | awk '{print $3}')
        READY_COUNT=$(kubectl -n kar-system get pods -l name="$1" -o wide | grep "$1" | awk '{print $2}' | awk -F / '{print $1}')
        if [[ "$POD_STATUS" == "Running" ]] && [[ "$READY_COUNT" != "0" ]]; then
            echo "$1 is ready."
            break
        fi
        echo "Waiting for $1 to be ready."
        kubectl get pods -n kar-system -o wide
        sleep 10
    done
}

if [ "$injectorOnly" == "" ]; then
    waitForPod "kar-redis"
    waitForPod "kar-kafka"
fi
waitForPod "kar-injector"
