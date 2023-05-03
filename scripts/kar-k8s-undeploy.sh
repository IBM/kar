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

# Script to automate removal of KAR runtime from a Kubernetes cluster

SCRIPTDIR=$(cd $(dirname "$0") && pwd)

echo "Undeploying KAR; deleting namespace may take a little while..."

$SCRIPTDIR/kar-k8s-namespace-disable.sh default

helm delete kar -n kar-system

if kubectl get secret kar.ibm.com.image-pull -n kar-system 2>/dev/null 1>/dev/null; then
    echo "Attempting to delete API Key kar-cr-reader-key from service account kar-cr-reader-id"
    ibmcloud iam service-api-key-delete kar-cr-reader-key kar-cr-reader-id -f
fi

kubectl delete ns kar-system
