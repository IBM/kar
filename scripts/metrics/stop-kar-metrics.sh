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

# change to metrics directory
cd $(dirname "$0")

NAMESPACE=prometheus

# Prometheus Server, Prometheus Operator, Grafana, K8s Dashboards
helm uninstall prometheus -n $NAMESPACE  2>/dev/null
# Redis dashboard
kubectl delete cm redis-dashboard-cm -n $NAMESPACE
# Redis
helm uninstall redis-exporter  -n $NAMESPACE  2>/dev/null
# Kar (ServiceMonitor + Service)

kubectl delete -f kafka-dash.yaml -n $NAMESPACE

kubectl delete -f kafka-exporter-service-monitor.yaml -n $NAMESPACE

kubectl delete ns $NAMESPACE
