#!/bin/bash

# This script creates a KIND cluster and deploys the nginx-based ingress
# controller on it.  This enables services running on the cluster to be
# exposed by creating Ingress instances.

# kind version that matches below tags
KIND_EXPECTED_VERSION=v0.9.0


# Valid node tags for kind 0.9.0
# Kubernetes 1.19: kindest/node:v1.19.1@sha256:98cf5288864662e37115e362b23e4369c8c4a408f99cbc06e58ac30ddc721600
# Kubernetes 1.18: kindest/node:v1.18.8@sha256:f4bcc97a0ad6e7abaf3f643d890add7efe6ee4ab90baeb374b4f41a4c95567eb
# Kubernetes 1.17: kindest/node:v1.17.11@sha256:5240a7a2c34bf241afb54ac05669f8a46661912eab05705d660971eeb12f6555
# Kubernetes 1.16: kindest/node:v1.16.15@sha256:a89c771f7de234e6547d43695c7ab047809ffc71a0c3b65aa54eda051c45ed20
# Kubernetes 1.15: kindest/node:v1.15.12@sha256:d9b939055c1e852fe3d86955ee24976cab46cba518abcb8b13ba70917e6547a6
KIND_NODE_TAG=${KIND_NODE_TAG:="v1.18.8@sha256:f4bcc97a0ad6e7abaf3f643d890add7efe6ee4ab90baeb374b4f41a4c95567eb"}

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
if [ "${running}" != 'true' ]; then
  docker network connect kind "${reg_name}"
fi

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
