#!/bin/bash

# Script to build a Camel HTTP integration locally and run it in Docker

SCRIPTDIR=../../scripts
WORKSPACE=workspace-http-integration-docker

# Clear any previous attempts

rm -rf $WORKSPACE

# Create workspace directory where all intermediate integration files will be stored

mkdir -p $WORKSPACE

# Build the integration locally

./$SCRIPTDIR/kamel-local-build.sh --workspace $WORKSPACE input.yaml

# Build the docker integration

./$SCRIPTDIR/kamel-docker-build.sh $WORKSPACE -t http-image

# Run the docker integration

./$SCRIPTDIR/kamel-docker-run.sh http-image
