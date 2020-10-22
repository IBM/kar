#!/bin/bash

# Script to containerize a kamel integration

SCRIPTDIR=$(cd $(dirname "$0") && pwd)

docker build -f "$SCRIPTDIR"/kamel/Dockerfile "$@"
