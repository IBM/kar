#!/bin/bash

# Automate namespace creation for KAR

KAR_NS=$1

if [ -z "$KAR_NS" ]; then
  echo "Usage: create-app-namespace.sh <namespace>"
  exit 1
fi

# create namespace
kubectl create namespace $KAR_NS

# copy secrets from kar-system
kubectl get secret kar.ibm.com.image-pull -n kar-system -o yaml | sed "s/kar-system/$KAR_NS/g" | kubectl -n $KAR_NS create -f -

kubectl get secret kar.ibm.com.runtime-config -n kar-system -o yaml | sed "s/kar-system/$KAR_NS/g" | kubectl -n $KAR_NS create -f -

# label namespace as KAR-enabled
kubectl label namespace $KAR_NS kar.ibm.com/enabled=true
