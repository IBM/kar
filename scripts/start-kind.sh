#!/bin/bash

# This script creates a KIND cluster and deploys the nginx-based ingress
# controller on it.  This enables services running on the cluster to be
# exposed by creating Ingress instances.

# kind version that matches below tags
KIND_EXPECTED_VERSION=v0.8.1

# Valid node tags for kind 0.8.1
# Kubernetes 1.18: v1.18.2@sha256:7b27a6d0f2517ff88ba444025beae41491b016bc6af573ba467b70c5e8e0d85f
# Kubernetes 1.17: v1.17.5@sha256:ab3f9e6ec5ad8840eeb1f76c89bb7948c77bbf76bcebe1a8b59790b8ae9a283a
# Kubernetes 1.16: v1.16.9@sha256:7175872357bc85847ec4b1aba46ed1d12fa054c83ac7a8a11f5c268957fd5765
# Kubernetes 1.15: v1.15.11@sha256:6cc31f3533deb138792db2c7d1ffc36f7456a06f1db5556ad3b6927641016f50
KIND_NODE_TAG=${KIND_NODE_TAG:="v1.16.9@sha256:7175872357bc85847ec4b1aba46ed1d12fa054c83ac7a8a11f5c268957fd5765"}

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
  docker exec ${node} sh -c "echo $(docker inspect --format '{{.NetworkSettings.IPAddress }}' registry) registry >> /etc/hosts"
  kubectl annotate node "${node}" "kind.x-k8s.io/registry=registry:5000";
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
