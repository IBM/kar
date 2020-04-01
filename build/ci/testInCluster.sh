#!/bin/bash

set -eu

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/../.."

helm install lt $ROOTDIR/examples/unit-tests/deploy/chart --set image=example-unit-tests:dev

if helm test lt; then
    echo "PASSED! In cluster unit-tests passed."
    helm delete lt
else
    echo "FAILED: In cluster unit-tests failed."
    kubectl logs ut-client -c client
    kubectl logs ut-client -c kar
    kubectl delete pod ut-client
    helm delete lt
    exit 1
fi


