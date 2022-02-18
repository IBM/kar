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

set -eu

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/.."

echo "*** Running in-cluster unit tests ***"
helm install ut $ROOTDIR/examples/unit-tests/deploy/chart --set image=localhost:5000/kar/kar-examples-js-unit-tests

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

helm install ykt $ROOTDIR/examples/actors-ykt/deploy/chart --set image=localhost:5000/kar/kar-examples-js-actors-ykt

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

helm install ykt-sc $ROOTDIR/examples/actors-ykt/deploy/chart --set image=localhost:5000/kar/kar-examples-js-actors-ykt --set noSidecar=true

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

echo "*** Running in-cluster actors-python ***"

helm install actors-py $ROOTDIR/examples/actors-python/deploy/chart --set image=localhost:5000/kar/kar-examples-actors-python-cluster

if helm test actors-py; then
    echo "PASSED! In cluster actors-python passed."
    helm delete actors-py
else
    echo "FAILED: In cluster actors-python failed."
    kubectl logs actor-client -c client
    kubectl logs actor-client -c kar
    kubectl delete pod actors-python-client
    helm delete actors-py
    exit 1
fi
