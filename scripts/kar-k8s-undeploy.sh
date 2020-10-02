#!/bin/bash

# Script to automate removal of KAR runtime from a Kubernetes cluster

echo "Undeploying KAR; deleting namespace may take a little while..."

helm delete kar -n kar-system

if kubectl get secret kar.ibm.com.image-pull -n kar-system 2>/dev/null 1>/dev/null; then
    echo "Attempting to delete API Key kar-cr-reader-key from service account kar-cr-reader-id"
    ibmcloud iam service-api-key-delete kar-cr-reader-key kar-cr-reader-id -f
fi

kubectl delete ns kar-system

