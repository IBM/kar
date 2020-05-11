#!/bin/bash
#
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

#
# This script is based on `deploy-chart.sh` from openwhisk-deploy-kube
# 

NAMESPACE=kar-system
TIMEOUT_STEP_LIMIT=30

#################
# Helper functions for verifying pod creation
#################

deploymentHealthCheck () {
  if [ -z "$1" ]; then
    echo "Error, component health check called without a component parameter"
    exit 1
  fi

  PASSED=false
  TIMEOUT=0
  until $PASSED || [ $TIMEOUT -eq $TIMEOUT_STEP_LIMIT ]; do
    KUBE_DEPLOY_STATUS=$(kubectl -n $NAMESPACE get pods -l name="$1" -o wide | grep "$1" | awk '{print $3}')
    KUBE_READY_COUNT=$(kubectl -n $NAMESPACE get pods -l name="$1" -o wide | grep "$1" | awk '{print $2}' | awk -F / '{print $1}')
    if [[ "$KUBE_DEPLOY_STATUS" == "Running" ]] && [[ "$KUBE_READY_COUNT" != "0" ]]; then
      PASSED=true
      echo "The deployment $1 is ready"
      break
    fi

    kubectl get pods -n $NAMESPACE -o wide

    let TIMEOUT=TIMEOUT+1
    sleep 10
  done

  if [ "$PASSED" == "false" ]; then
    echo "Failed to finish deploying $1"

    kubectl -n $NAMESPACE logs $(kubectl -n $NAMESPACE get pods -l name="$1" -o wide | grep "$1" | awk '{print $1}')
    exit 1
  fi

  echo "$1 is up and running"
}

statefulsetHealthCheck () {
  if [ -z "$1" ]; then
    echo "Error, StatefulSet health check called without a parameter"
    exit 1
  fi

  PASSED=false
  TIMEOUT=0
  until $PASSED || [ $TIMEOUT -eq $TIMEOUT_STEP_LIMIT ]; do
    KUBE_DEPLOY_STATUS=$(kubectl -n $NAMESPACE get pods -l name="$1" -o wide | grep "$1"-0 | awk '{print $3}')
    KUBE_READY_COUNT=$(kubectl -n $NAMESPACE get pods -l name="$1" -o wide | grep "$1"-0 | awk '{print $2}' | awk -F / '{print $1}')
    if [[ "$KUBE_DEPLOY_STATUS" == "Running" ]] && [[ "$KUBE_READY_COUNT" != "0" ]]; then
      PASSED=true
      echo "The statefulset $1 is ready"
      break
    fi

    kubectl get pods -n $NAMESPACE -o wide

    let TIMEOUT=TIMEOUT+1
    sleep 10
  done

  if [ "$PASSED" == "false" ]; then
    echo "Failed to finish deploying $1"
    # Dump all namespaces in case the problem is with a pod in the kube-system namespace
    kubectl get pods --all-namespaces -o wide

    kubectl -n $NAMESPACE logs $(kubectl -n $NAMESPACE get pods -o wide | grep "$1"-0 | awk '{print $1}')
    exit 1
  fi

  echo "$1-0 is up and running"

}

jobHealthCheck () {
  if [ -z "$1" ]; then
    echo "Error, job health check called without a component parameter"
    exit 1
  fi

  PASSED=false
  TIMEOUT=0
  until $PASSED || [ $TIMEOUT -eq $TIMEOUT_STEP_LIMIT ]; do
    KUBE_DEPLOY_STATUS=$(kubectl -n $NAMESPACE get pods -l name="$1" -o wide | grep "$1" | awk '{print $3}')
    if [[ $KUBE_DEPLOY_STATUS == *Completed* ]]; then
      PASSED=true
      echo "The job $1 has completed"
      break
    fi

    kubectl get pods -n $NAMESPACE -o wide

    let TIMEOUT=TIMEOUT+1
    sleep 10
  done

  if [ "$PASSED" == "false" ]; then
    echo "Failed to finish running $1"
    # Dump all namespaces in case the problem is with a pod in the kube-system namespace
    kubectl get pods --all-namespaces -o wide

    kubectl -n $NAMESPACE logs jobs/$1
    exit 1
  fi

  echo "$1 completed"
}


