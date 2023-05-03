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

# Script to automate installation of Prometheus stack to expose  metrics

#start metrics node if needed
check=$(k3d node list | grep metrics-master | awk '{print $1}')
if [ -z $check ];
then
    echo "starting metrics node"
    k3d node create metrics-master --wait
    kubectl label nodes k3d-metrics-master-0 metrics-type=master
fi

NAMESPACE=prometheus
set -e

# change to metrics directory
cd $(dirname "$0")

nsStatus=$(kubectl get ns $NAMESPACE -o json 2>/dev/null | jq .status.phase -r)
if [[ $nsStatus != "Active" ]]
then
   echo ".... Creating prometheus namespace"
   kubectl create ns prometheus
fi

helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update 2>/dev/null

echo "... Wait, installing Prometheus, Grafana, and other goodies ..."

# Prometheus Server, Prometheus Operator, Grafana, Node Exporter, K8s Dashboards
helm install prometheus prometheus-community/kube-prometheus-stack -f prom-overrides.yaml --wait -n $NAMESPACE  2>/dev/null

# Redis
helm install redis-exporter -f redis-values.yaml prometheus-community/prometheus-redis-exporter --wait -n $NAMESPACE  2>/dev/null

# Redis dashboard
kubectl create -f redis-dashboard-cm.yaml -n $NAMESPACE

# grafana dashboard for kafka exporter
kubectl create -f kafka-dash.yaml -n $NAMESPACE

kubectl create -f kafka-exporter-service-monitor.yaml

echo "To view metrics in Grafana:"
echo "  1. kubectl port-forward -n prometheus svc/prometheus-grafana  3000:80"
echo "  2. Point your browser to localhost:3000"
echo "  3. login with default user/password: admin/prom-operator"
