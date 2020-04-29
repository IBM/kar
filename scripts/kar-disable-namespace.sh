#!/bin/bash

# Automate namespace disablement for KAR

KAR_NS=$1

if [ -z "$KAR_NS" ]; then
  echo "Usage: kar-disable-namespace.sh <namespace>"
  exit 1
fi

# delete secrets
kubectl -n $KAR_NS delete secret kar.ibm.com.image-pull 2>/dev/null
kubectl -n $KAR_NS delete secret kar.ibm.com.runtime-config 2>/dev/null

# label namespace as not KAR-enabled
kubectl label namespace $KAR_NS kar.ibm.com/enabled=false --overwrite
