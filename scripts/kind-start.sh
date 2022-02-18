#!/bin/bash

#
# Copyright IBM Corporation 2020,2022
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

# This script creates a KIND cluster and deploys the nginx-based ingress
# controller on it.  This enables services running on the cluster to be
# exposed by creating Ingress instances.

# kind version that matches below tags
KIND_EXPECTED_VERSION=v0.11.1

# Valid node tags for kind 0.11.1
# v1.21.1@sha256:69860bda5563ac81e3c0057d654b5253219618a22ec3a346306239bba8cfa1a6
# v1.20.7@sha256:cbeaf907fc78ac97ce7b625e4bf0de16e3ea725daf6b04f930bd14c67c671ff9
# v1.19.11@sha256:07db187ae84b4b7de440a73886f008cf903fcf5764ba8106a9fd5243d6f32729
# v1.18.19@sha256:7af1492e19b3192a79f606e43c35fb741e520d195f96399284515f077b3b622c
# v1.17.17@sha256:66f1d0d91a88b8a001811e2f1054af60eef3b669a9a74f9b6db871f2f1eeed00
# v1.16.15@sha256:83067ed51bf2a3395b24687094e283a7c7c865ccc12a8b1d7aa673ba0c5e8861
# v1.15.12@sha256:b920920e1eda689d9936dfcf7332701e80be12566999152626b2c9d730397a95
# v1.14.10@sha256:f8a66ef82822ab4f7569e91a5bccaf27bceee135c1457c512e54de8c6f7219f8
KIND_NODE_TAG=${KIND_NODE_TAG:="v1.20.7@sha256:cbeaf907fc78ac97ce7b625e4bf0de16e3ea725daf6b04f930bd14c67c671ff9"}

set -eu

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/.."

# Validate kind version is as expected
KIND_ACTUAL_VERSION=$(kind version | awk '/ /{print $2}')
if [ "$KIND_ACTUAL_VERSION" != "$KIND_EXPECTED_VERSION" ]; then
    echo "Kind version mismatch: expected $KIND_EXPECTED_VERSION but found $KIND_ACTUAL_VERSION"
    exit 1
fi

# create registry container unless it already exists
reg_name='registry'
running="$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)"
if [ "${running}" != 'true' ]; then
  docker run -d --restart=always -p 5000:5000 --name registry registry:2
fi

# Boot cluster
kind create cluster --config ${SCRIPTDIR}/kind/kind-cluster.yaml --name kind --image kindest/node:${KIND_NODE_TAG} --wait 10m || exit 1

########
# Begin script from https://kind.sigs.k8s.io/docs/user/local-registry/
########

# Connect the registry to the cluster network
docker network connect kind "${reg_name}" || true

for node in $(kind get nodes --name kind); do
  kubectl annotate node "${node}" "kind.x-k8s.io/registry=localhost:5000";
done

########
# End of script from https://kind.sigs.k8s.io/docs/user/local-registry/
########


echo "KIND cluster is running and reachable"
kubectl get nodes

# Deploy nginx-ingress controller
kubectl apply -f ${SCRIPTDIR}/kind/ingress-nginx/mandatory.yaml
kubectl apply -f ${SCRIPTDIR}/kind/ingress-nginx/service-nodeport.yaml

echo "Ingress controller deployed"
kubectl get all -n ingress-nginx
