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

# Script to automate installation of Prometheus stack to expose  metrics 

NAMESPACE=prometheus
set -e

SCRIPTDIR=$(cd $(dirname "$0") && pwd)

nsStatus=$(kubectl get ns $NAMESPACE -o json 2>/dev/null | jq .status.phase -r)
if [[ $nsStatus != "Active" ]]
then
   echo ".... Creating prometheus namespace"
   kubectl create ns prometheus
fi

helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update 2>/dev/null

# Prometheus Server, Prometheus Operator, Grafana, Node Exporter, K8s Dashboards
helm install prometheus prometheus-community/kube-prometheus-stack -f $SCRIPTDIR/metrics/prom-overrides.yaml --wait -n $NAMESPACE  2>/dev/null

# Redis
helm install redis-exporter -f $SCRIPTDIR/metrics/redis-values.yaml prometheus-community/prometheus-redis-exporter --wait -n $NAMESPACE  2>/dev/null

# Redis dashboard
kubectl create -f $SCRIPTDIR/metrics/redis-dashboard-cm.yaml -n $NAMESPACE

# Kar (ServiceMonitor + Service)
kubectl create -f $SCRIPTDIR/metrics/kar-metrics.yaml -n $NAMESPACE

# grafana dashboard for kafka exporter
kubectl create -f $SCRIPTDIR/metrics/kafka-dash.yaml -n $NAMESPACE

kubectl create -f $SCRIPTDIR/metrics/kafka-exporter-service-monitor.yaml
