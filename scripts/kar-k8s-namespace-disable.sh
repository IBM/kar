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

# Automate namespace disablement for KAR

KAR_NS=$1

if [ -z "$KAR_NS" ]; then
  echo "Usage: kar-k8s-namespace-disable.sh <namespace>"
  exit 1
fi

# delete secrets
kubectl -n $KAR_NS delete secret kar.ibm.com.image-pull 2>/dev/null
kubectl -n $KAR_NS delete secret kar.ibm.com.runtime-config 2>/dev/null

# label namespace as not KAR-enabled
kubectl label namespace $KAR_NS kar.ibm.com/enabled=false --overwrite
