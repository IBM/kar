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

#!/bin/bash

NAMESPACE=logging

kctl() {
    kubectl --namespace "$NAMESPACE" "$@"
}

# change to logging directory
cd $(dirname "$0")


kctl delete -f /tmp/elasticsearch_statefulset.yaml

kctl delete -f kibana/kibana.yaml

kctl delete -f fluentd/aggregator-cm.yaml

kctl delete -f fluentd/aggregator.yaml 2>/dev/null

kctl delete -f fluentd/collector-cm.yaml 

kctl delete -f fluentd/collector.yaml 2>/dev/null

kubectl delete ns ${NAMESPACE}
