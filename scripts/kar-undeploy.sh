#!/bin/bash

# Script to automate removal of KAR runtime from a Kubernetes cluster

echo "Undeploying KAR; deleting namespace may take a little while..."

helm delete kar -n kar-system

kubectl delete ns kar-system

