#!/bin/bash

set -eu

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/../.."
cd $ROOTDIR

. $SCRIPTDIR/kube-helpers.sh

kubectl create namespace kar-system

helm install kar charts/kar -n kar-system -f $SCRIPTDIR/kar-config.yaml

statefulsetHealthCheck "kar-redis"
statefulsetHealthCheck "kar-kafka"
deploymentHealthCheck "kar-injector"
