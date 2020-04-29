#!/bin/bash

# Automate namespace enablement for KAR

KAR_NS=$1

if [ -z "$KAR_NS" ]; then
  echo "Usage: kar-enable-namespace.sh <namespace>"
  exit 1
fi

# create namespace if it doesn't already exist
if ! kubectl get namespace $KAR_NS 2>/dev/null 1>/dev/null; then
    kubectl create namespace $KAR_NS
fi

# copy secrets from kar-system
if kubectl get secret kar.ibm.com.image-pull -n kar-system 2>/dev/null 1>/dev/null; then
    kubectl get secret kar.ibm.com.image-pull -n kar-system -o yaml | sed "s/kar-system/$KAR_NS/g" | kubectl -n $KAR_NS create -f -
fi

kubectl get secret kar.ibm.com.runtime-config -n kar-system -o yaml | sed "s/kar-system/$KAR_NS/g" | kubectl -n $KAR_NS create -f -

# label namespace as KAR-enabled
kubectl label namespace $KAR_NS kar.ibm.com/enabled=true --overwrite
