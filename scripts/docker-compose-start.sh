#!/bin/bash

# Script to launch the KAR runtime using docker-compose

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
cd $SCRIPTDIR

# Create bridge network for use by kar runtime system
if [ -z $(docker network ls --filter name=kar-bus --format '{{.Name}}') ]; then
    docker network create kar-bus
fi

docker-compose up -d
