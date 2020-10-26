#!/bin/bash

# Script to build a Camel HTTP integration locally

SCRIPTDIR=../../scripts
WORKSPACE=workspace-http-integration

# Clear any previous attempts

rm -rf $WORKSPACE

# Create workspace directory where all intermediate integration files will be stored

mkdir -p $WORKSPACE

# Build the integration locally

./$SCRIPTDIR/kamel-local-build.sh --workspace $WORKSPACE input.yaml

# Run the integration locally

./$SCRIPTDIR/kamel-local-run.sh --workspace $WORKSPACE
