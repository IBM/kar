#!/bin/bash

# Script to automate installation of KAR runtime into a Kubernetes cluster

set -e

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/.."

. $ROOTDIR/build/ci/kube-helpers.sh

help=""
args=""
parse=true
apikey=""
helmargs=""
while [ -n "$1" ]; do
    if [ -z "$parse" ]; then
        args="$args '$1'"
        shift
        continue
    fi
    case "$1" in
        -h|-help|--help) help="1"; break;;
        -a|-apikey|--apikey) shift; apikey="$1";;
        -d|-dev|--dev) helmargs="$helmargs -f $ROOTDIR/build/ci/kar-dev.yaml";;
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
    -a -apikey <apikey>       install <apikey> as KAR image pull secret
    -f -file <config.yaml>    pass `-f <config.yaml>` to `helm install kar`
    -s -set key=value         pass `--set key=value to `helm install kar`
    -d -dev                   pass `-f kar-dev.yaml` to helm install kar`
EOF
    exit 0
fi

cd $ROOTDIR

kubectl create namespace kar-system || true

if [ -n "$apikey" ]; then
    kubectl --namespace kar-system create secret docker-registry kar.ibm.com.image-pull --docker-server=us.icr.io --docker-username=iamapikey --docker-email=kar@ibm.com --docker-password=$apikey
fi

helm install kar charts/kar -n kar-system $helmargs

statefulsetHealthCheck "kar-redis"
statefulsetHealthCheck "kar-kafka"
deploymentHealthCheck "kar-injector"
