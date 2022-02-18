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

# kar version level required for compatibility
K3D_REQUIRED_VERSION=v5

#exit on script errors and unset variables
set -ue
#set -x

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/.."
cd $SCRIPTDIR

# Validate k3d version is usable
K3D_ACTUAL_VERSION=$(k3d version | head -1 | awk '/ /{print $3}')
if [[ $K3D_ACTUAL_VERSION != ${K3D_REQUIRED_VERSION}* ]]; then
    echo "K3d version problem: need compatible $K3D_REQUIRED_VERSION but found $K3D_ACTUAL_VERSION"
    exit 1
fi

# create registry container unless it already exists
reg_name='registry'
running="$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)"
if [ "${running}" != 'true' ]; then
    docker run -d --restart=always -p 5000:5000 --name registry registry:2
fi

# Boot cluster
k3d cluster create -p "31080:80@loadbalancer" --registry-config $(pwd)/k3d/registries.yaml --k3s-arg "--disable=traefik@server:0" --volume "$(pwd)/k3d/helm-ingress-nginx.yaml:/var/lib/rancher/k3s/server/manifests/helm-ingress-nginx.yaml"

# make sure registry is connected to k3d network
connected="not"$({ docker network inspect k3d-k3s-default | grep -e '"Name": "registry"' || true; })
if [ "not" == "$connected" ]; then
    docker network connect k3d-k3s-default registry
fi

# TODO check if affinity is required; For now, assume yes
# TODO check if worker nodes are needed; For now, assume yes

# wait for ingres to be ready before creating worker nodes
printf "waiting for ingress-controller-nginx to be ready: "
while [[ $(kubectl get po -l app.kubernetes.io/name=ingress-nginx -n kube-system -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]]; do printf "." && sleep 5; done
echo 

# create 2 worker nodes and kar system node
k3d node create workernode-1 --wait
k3d node create workernode-2 --wait
k3d node create karsystemnode --wait

# label the new worker and kar system node
kubectl label nodes k3d-karsystemnode-0 kar-type=system
kubectl label nodes k3d-workernode-1-0 kar-type=worker
kubectl label nodes k3d-workernode-2-0 kar-type=worker

# tell em what they got
kubectl cluster-info
kubectl get nodes --show-labels

# tell em how to get rid of it
echo
echo "to remove cluster:  k3d cluster delete"

