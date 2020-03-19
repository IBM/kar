#!/bin/bash

# This script creates a KIND cluster and deploys the nginx-based ingress
# controller on it.  This enables services running on the cluster to be
# exposed by creating Ingress instances.

TRAVIS_KUBE_VERSION=v1.16.4

set -x

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/../.."

# Boot cluster
kind create cluster --config ${SCRIPTDIR}/kind-cluster.yaml --name kind --image kindest/node:${TRAVIS_KUBE_VERSION} --wait 10m || exit 1

echo "KIND cluster is running and reachable"
kubectl get nodes

# Deploy nginx-ingress controller
kubectl apply -f ${SCRIPTDIR}/ingress-nginx/mandatory.yaml
kubectl apply -f ${SCRIPTDIR}/ingress-nginx/service-nodeport.yaml

echo "Ingress controller deployed"
kubectl get all -n ingress-nginx
