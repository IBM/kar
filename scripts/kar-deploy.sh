#!/bin/bash

# Script to automate installation of KAR runtime into a Kubernetes cluster

set -e

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/.."

. $ROOTDIR/build/ci/kube-helpers.sh

help=""
args=""
parse=true
icr="enabled"
helmargs=""
while [ -n "$1" ]; do
    if [ -z "$parse" ]; then
        args="$args '$1'"
        shift
        continue
    fi
    case "$1" in
        -h|-help|--help) help="1"; break;;
        -d|-dev|--dev)
            helmargs="$helmargs -f $ROOTDIR/build/ci/kar-dev.yaml"
            icr="disabled"
            ;;
        -f|-file|--file) shift; helmargs="$helmargs -f $1";;
        -s|-set|--set) shift; helmargs="$helmargs --set $1";;
        --) parse=;;
        *) args="$args '$1'";;
    esac
    shift
done

if [ -n "$help" ]; then
    cat << EOF
Usage: kay-deploy.sh [options]
where [options] includes:
    -f -file <config.yaml>    pass `-f <config.yaml>` to `helm install kar`
    -s -set key=value         pass `--set key=value to `helm install kar`
    -d -dev                   pass `-f kar-dev.yaml` to helm install kar`
EOF
    exit 0
fi

cd $ROOTDIR

kubectl create namespace kar-system 2>/dev/null || true

if [ "$icr" == "enabled" ]; then
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

    kubectl --namespace kar-system create secret docker-registry kar.ibm.com.image-pull --docker-server=us.icr.io --docker-username=iamapikey --docker-email=kar@ibm.com --docker-password=$KAR_API_KEY
fi

helm install kar charts/kar -n kar-system $helmargs

statefulsetHealthCheck "kar-redis"
statefulsetHealthCheck "kar-kafka"
deploymentHealthCheck "kar-injector"
