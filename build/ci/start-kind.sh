#!/bin/bash

set -x

TRAVIS_KUBE_VERSION=v1.16.4

# Create cluster config
cat > mycluster.yaml <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker
EOF

# Boot cluster
kind create cluster --config mycluster.yaml --name kind --image kindest/node:${TRAVIS_KUBE_VERSION} --wait 10m || exit 1

echo "Kubernetes cluster is deployed and reachable"
kubectl describe nodes
