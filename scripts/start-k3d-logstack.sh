#
# Copyright IBM Corporation 2020,2021
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

nsStatus=$(kubectl get ns $NAMESPACE -o json 2>/dev/null | jq .status.phase -r)
if [[ $nsStatus != "Active" ]]
then
   echo ".... Creating logging namespace"
   kubectl create namespace "$NAMESPACE"
fi

kctl() {
    kubectl --namespace "$NAMESPACE" "$@"
}
#  define vars for elasticsearch_statefulset.template.yaml below
if [ -z "${ES_DATA_DIR}" ]; then
   ES_DATA_DIR=/usr/share/elasticsearch/data
fi
if [ -z "${ES_STORAGE_SIZE}" ]; then
   ES_STORAGE_SIZE=5Gi
fi

# --------------------------------------------------------
# deploy elasticsearch service
#---------------------------------------------------------
kctl create -f logging/elasticsearch/elasticsearch_svc.yaml
#-----------------------------------------------------------------------------------------------------
# use elasticsearch template to customize deployment properties like data dir, size of the ES data volume 
#----------------------------------------------------------------------------------------------------
sed < logging/elasticsearch/template/elasticsearch_statefulset.template.yaml -e "s|{{data-dir}}|$ES_DATA_DIR|g" -e "s|{{es-storage-size}}|$ES_STORAGE_SIZE|g" > /tmp/elasticsearch_statefulset.yaml
# -----------------------------------------------------------
# ELASTICSEARCH
# ----------------------------------------------------------
kctl create -f /tmp/elasticsearch_statefulset.yaml
kctl rollout status sts/elasticsearch
#
# Test if ES deploy:
# kubectl port-forward es-cluster-0 9200:9200 --namespace=logging
# curl -X GET "localhost:9200/_cluster/health?pretty"
#
# ------------------------------
# KIBANA
# -----------------------------
kctl create -f logging/kibana/kibana.yaml
kctl rollout status deployment/kibana 
# ----------------------------------------------------------------------------------
# Deploy fluentd aggregator config map. It contains fluentd aggregator configuration
# ---------------------------------------------------------------------------------
kctl create -f logging/fluentd/aggregator-cm.yaml
# ----------------------------------------------
# FLUENTD AGGREGATOR (deploys onlastic search master node)
# ----------------------------------------------
kctl create -f logging/fluentd/aggregator.yaml
kctl rollout status daemonset/fluentd-agg 
# ----------------------------------------------
# Deploy service in front of the aggregate so that
# the collector can connect to it by name
#----------------------------------------------
kctl create -f logging/fluentd/fluentd-agg-svc.yaml
# --------------------------------------------------------------------------------
# Deploy fluentd collector config map. It contains fluentd collector configuration
# --------------------------------------------------------------------------------
kctl create -f logging/fluentd/collector-cm.yaml
# ---------------------------------------
# FLUENTD COLLECTOR (one per worker node)
# ---------------------------------------
kctl create -f logging/fluentd/collector.yaml
kctl rollout status daemonset/fluentd-col 
echo "EFK log stack deployed"
echo "Use port-forward to access Kibana GUI: kubectl port-forward svc/kibana  5602:5601 --namespace=logging"
