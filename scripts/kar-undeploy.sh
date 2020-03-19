#!/bin/bash

# Script to automate removal of KAR runtime from a Kubernetes cluster

helm delete kar -n kar-system

kubectl delete secret kar.ibm.com.image-pull -n kar-system

kubectl delete ns kar-system

