#!/bin/bash

#
# Copyright IBM Corporation 2020,2023
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

set -eu

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/.."

# Valid node tags for kind 0.20.0
# 1.27: kindest/node:v1.27.3@sha256:3966ac761ae0136263ffdb6cfd4db23ef8a83cba8a463690e98317add2c9ba72
# 1.26: kindest/node:v1.26.6@sha256:6e2d8b28a5b601defe327b98bd1c2d1930b49e5d8c512e1895099e4504007adb
# 1.25: kindest/node:v1.25.11@sha256:227fa11ce74ea76a0474eeefb84cb75d8dad1b08638371ecf0e86259b35be0c8
# 1.24: kindest/node:v1.24.15@sha256:7db4f8bea3e14b82d12e044e25e34bd53754b7f2b0e9d56df21774e6f66a70ab
# 1.23: kindest/node:v1.23.17@sha256:59c989ff8a517a93127d4a536e7014d28e235fb3529d9fba91b3951d461edfdb
# 1.22: kindest/node:v1.22.17@sha256:f5b2e5698c6c9d6d0adc419c0deae21a425c07d81bbf3b6a6834042f25d4fba2
# 1.21: kindest/node:v1.21.14@sha256:8a4e9bb3f415d2bb81629ce33ef9c76ba514c14d707f9797a01e3216376ba093

# Valid node tags for kind 0.19.0
# 1.27: kindest/node:v1.27.1@sha256:b7d12ed662b873bd8510879c1846e87c7e676a79fefc93e17b2a52989d3ff42b
# 1.26: kindest/node:v1.26.4@sha256:f4c0d87be03d6bea69f5e5dc0adb678bb498a190ee5c38422bf751541cebe92e
# 1.25: kindest/node:v1.25.9@sha256:c08d6c52820aa42e533b70bce0c2901183326d86dcdcbedecc9343681db45161
# 1.24: kindest/node:v1.24.13@sha256:cea86276e698af043af20143f4bf0509e730ec34ed3b7fa790cc0bea091bc5dd
# 1.23: kindest/node:v1.23.17@sha256:f77f8cf0b30430ca4128cc7cfafece0c274a118cd0cdb251049664ace0dee4ff
# 1.22: kindest/node:v1.22.17@sha256:9af784f45a584f6b28bce2af84c494d947a05bd709151466489008f80a9ce9d5
# 1.21: kindest/node:v1.21.14@sha256:220cfafdf6e3915fbce50e13d1655425558cb98872c53f802605aa2fb2d569cf

# Valid node tags for kind 0.18.0
# 1.26: kindest/node:v1.26.3@sha256:61b92f38dff6ccc29969e7aa154d34e38b89443af1a2c14e6cfbd2df6419c66f
# 1.25: kindest/node:v1.25.8@sha256:00d3f5314cc35327706776e95b2f8e504198ce59ac545d0200a89e69fce10b7f
# 1.24: kindest/node:v1.24.12@sha256:1e12918b8bc3d4253bc08f640a231bb0d3b2c5a9b28aa3f2ca1aee93e1e8db16
# 1.23: kindest/node:v1.23.17@sha256:e5fd1d9cd7a9a50939f9c005684df5a6d145e8d695e78463637b79464292e66c
# 1.22: kindest/node:v1.22.17@sha256:c8a828709a53c25cbdc0790c8afe12f25538617c7be879083248981945c38693
# 1.21: kindest/node:v1.21.14@sha256:27ef72ea623ee879a25fe6f9982690a3e370c68286f4356bf643467c552a3888

# Valid node tags for kind 0.17.0
# 1.25: kindest/node:v1.25.3@sha256:f52781bc0d7a19fb6c405c2af83abfeb311f130707a0e219175677e366cc45d1
# 1.24: kindest/node:v1.24.7@sha256:577c630ce8e509131eab1aea12c022190978dd2f745aac5eb1fe65c0807eb315
# 1.23: kindest/node:v1.23.13@sha256:ef453bb7c79f0e3caba88d2067d4196f427794086a7d0df8df4f019d5e336b61
# 1.22: kindest/node:v1.22.15@sha256:7d9708c4b0873f0fe2e171e2b1b7f45ae89482617778c1c875f1053d4cef2e41
# 1.21: kindest/node:v1.21.14@sha256:9d9eb5fb26b4fbc0c6d95fa8c790414f9750dd583f5d7cee45d92e8c26670aa1
# 1.20: kindest/node:v1.20.15@sha256:a32bf55309294120616886b5338f95dd98a2f7231519c7dedcec32ba29699394
# 1.19: kindest/node:v1.19.16@sha256:476cb3269232888437b61deca013832fee41f9f074f9bed79f57e4280f7c48b7

# Based on the kind version found in the path, select the correct Kubernetes 1.24 node tag
KIND_ACTUAL_VERSION=$(kind version | awk '/ /{print $2}')
case $KIND_ACTUAL_VERSION in
    v0.20.0)
        KIND_NODE_TAG=${KIND_NODE_TAG:="v1.24.15@sha256:7db4f8bea3e14b82d12e044e25e34bd53754b7f2b0e9d56df21774e6f66a70ab"}
        ;;
    v0.19.0)
        KIND_NODE_TAG=${KIND_NODE_TAG:="v1.24.13@sha256:cea86276e698af043af20143f4bf0509e730ec34ed3b7fa790cc0bea091bc5dd"}
        ;;
    v0.18.0)
        KIND_NODE_TAG=${KIND_NODE_TAG:="v1.24.12@sha256:1e12918b8bc3d4253bc08f640a231bb0d3b2c5a9b28aa3f2ca1aee93e1e8db16"}
        ;;
    v0.17.0)
        KIND_NODE_TAG=${KIND_NODE_TAG:="v1.24.7@sha256:577c630ce8e509131eab1aea12c022190978dd2f745aac5eb1fe65c0807eb315"}
        ;;
    *)
        echo "Unexpected kind version: $KIND_ACTUAL_VERSION"
    exit 1
esac

# create docker network "kind" so that rootless podman can connect registry
docker network create kind >/dev/null || true 

# create registry container unless it already exists
reg_name='registry'
reg_port='5000'
running="$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)"
if [ "${running}" != 'true' ]; then
  docker run -d --restart=always -p "127.0.0.1:${reg_port}:5000" --net=kind --name registry registry:2
fi

# Boot cluster
kind create cluster --config ${SCRIPTDIR}/kind/kind-cluster.yaml --name kind --image kindest/node:${KIND_NODE_TAG} --wait 10m || exit 1

########
# Begin script from https://kind.sigs.k8s.io/docs/user/local-registry/
########

# connect the registry to the cluster network if not already connected
if [ "$(docker inspect -f='{{json .NetworkSettings.Networks.kind}}' "${reg_name}")" = 'null' ]; then
  docker network connect "kind" "${reg_name}"
fi

# Document the local registry
# https://github.com/kubernetes/enhancements/tree/master/keps/sig-cluster-lifecycle/generic/1755-communicating-a-local-registry
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:${reg_port}"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF

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
