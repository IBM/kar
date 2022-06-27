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
KIND_EXPECTED_VERSION=v0.14.0

# Valid node tags for kind 0.14.0
# 1.24: kindest/node:v1.24.0@sha256:0866296e693efe1fed79d5e6c7af8df71fc73ae45e3679af05342239cdc5bc8e
# 1.23: kindest/node:v1.23.6@sha256:b1fa224cc6c7ff32455e0b1fd9cbfd3d3bc87ecaa8fcb06961ed1afb3db0f9ae
# 1.22: kindest/node:v1.22.9@sha256:8135260b959dfe320206eb36b3aeda9cffcb262f4b44cda6b33f7bb73f453105
# 1.21: kindest/node:v1.21.12@sha256:f316b33dd88f8196379f38feb80545ef3ed44d9197dca1bfd48bcb1583210207
# 1.20: kindest/node:v1.20.15@sha256:6f2d011dffe182bad80b85f6c00e8ca9d86b5b8922cdf433d53575c4c5212248
# 1.19: kindest/node:v1.19.16@sha256:d9c819e8668de8d5030708e484a9fdff44d95ec4675d136ef0a0a584e587f65c
# 1.18: kindest/node:v1.18.20@sha256:738cdc23ed4be6cc0b7ea277a2ebcc454c8373d7d8fb991a7fcdbd126188e6d7

# Valid node tags for kind 0.12.0
# v1.23.4@sha256:0e34f0d0fd448aa2f2819cfd74e99fe5793a6e4938b328f657c8e3f81ee0dfb9
# v1.22.7@sha256:1dfd72d193bf7da64765fd2f2898f78663b9ba366c2aa74be1fd7498a1873166
# v1.21.10@sha256:84709f09756ba4f863769bdcabe5edafc2ada72d3c8c44d6515fc581b66b029c
# v1.20.15@sha256:393bb9096c6c4d723bb17bceb0896407d7db581532d11ea2839c80b28e5d8deb
# v1.19.16@sha256:81f552397c1e6c1f293f967ecb1344d8857613fb978f963c30e907c32f598467
# v1.18.20@sha256:e3dca5e16116d11363e31639640042a9b1bd2c90f85717a7fc66be34089a8169
# v1.17.17@sha256:e477ee64df5731aa4ef4deabbafc34e8d9a686b49178f726563598344a3898d5
# v1.16.15@sha256:64bac16b83b6adfd04ea3fbcf6c9b5b893277120f2b2cbf9f5fa3e5d4c2260cc
# v1.15.12@sha256:9dfc13db6d3fd5e5b275f8c4657ee6a62ef9cb405546664f2de2eabcfd6db778
# v1.14.10@sha256:b693339da2a927949025869425e20daf80111ccabf020d4021a23c00bae29d82

KIND_NODE_TAG=${KIND_NODE_TAG:="v1.20.15@sha256:6f2d011dffe182bad80b85f6c00e8ca9d86b5b8922cdf433d53575c4c5212248"}

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
