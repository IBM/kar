#!/bin/bash

# Automate code-engine project disablement for KAR

ibmcloud code-engine secret delete -f --name kar.ibm.com.runtime-config

ibmcloud code-engine registry delete -f --name kar.ibm.com.image-pull
