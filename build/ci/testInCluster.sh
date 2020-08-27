#!/bin/bash

set -eu

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/../.."

echo "*** Running in-cluster unit tests ***"
helm install ut $ROOTDIR/examples/unit-tests/deploy/chart --set image=localhost:5000/examples/js/unit-tests

if helm test ut; then
    echo "PASSED! In cluster unit-tests passed."
    helm delete ut
else
    echo "FAILED: In cluster unit-tests failed."
    kubectl logs ut-client -c client
    kubectl logs ut-client -c kar
    kubectl delete pod ut-client
    helm delete ut
    exit 1
fi

echo "*** Running in-cluster actors-ykt ***"

helm install ykt $ROOTDIR/examples/actors-ykt/deploy/chart --set image=localhost:5000/examples/js/actors-ykt

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


echo "*** Running in-cluster no-sidecar actors-ykt ***"

helm install ykt-sc $ROOTDIR/examples/actors-ykt/deploy/chart --set image=localhost:5000/examples/js/actors-ykt --set localMode=true

if helm test ykt-sc; then
    echo "PASSED! In cluster no-sidecar actors-ykt passed."
    helm delete ykt-sc
else
    echo "FAILED: In cluster no-sidecar actors-ykt failed."
    kubectl logs ykt-client -c client
    kubectl logs ykt-client -c kar
    kubectl delete pod ykt-client
    helm delete ykt-sc
    exit 1
fi
