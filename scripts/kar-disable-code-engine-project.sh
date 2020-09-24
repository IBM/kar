#!/bin/bash

# Automate code-engine project disablement for KAR

ibmcloud code-engine secret delete --name kar.ibm.com.runtime-config
