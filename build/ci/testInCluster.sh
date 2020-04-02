#!/bin/bash

set -eu

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/../.."

echo "*** Running in-cluster unit tests ***"
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

echo "*** Running in-cluster actors-ykt ***"

helm install ykt $ROOTDIR/examples/actors-ykt/deploy/chart --set image=example-ykt:dev

if helm test ykt; then
    echo "PASSED! In cluster actors-ykt passed."
    helm delete ykt
else
    echo "FAILED: In cluster actors-ykt failed."
    kubectl logs ykt-client -c client
    kubectl logs ykt-client -c kar
    kubectl delete pod ykt-client
    helm delete ykt
    exit 1
fi


