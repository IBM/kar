#!/bin/bash

# Automate namespace disablement for KAR

KAR_NS=$1

if [ -z "$KAR_NS" ]; then
  echo "Usage: kar-disable-namespace.sh <namespace>"
  exit 1
fi

# delete secrets
kubectl -n $KAR_NS delete secret kar.ibm.com.image-pull
kubectl -n $KAR_NS delete secret kar.ibm.com.runtime-config

# label namespace as not KAR-enabled
kubectl label namespace $KAR_NS kar.ibm.com/enabled=false --overwrite
