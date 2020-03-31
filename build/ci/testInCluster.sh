#!/bin/bash

set -eu

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/../.."

helm install lt $ROOTDIR/examples/incr/deploy/charts/testHarness --set image=sample-incr:dev

if helm test lt; then
    echo "PASSED! In cluster incr/testHarness passed."
    helm delete lt
else
    echo "FAILED: In cluster incr/testHarness failed."
    kubectl logs incr-client -c client
    kubectl logs incr-client -c kar
    kubectl delete pod incr-client
    helm delete lt
    exit 1
fi


