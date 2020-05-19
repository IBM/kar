#!/bin/bash

# Script to launch kafka and redis using docker-compose

SCRIPTDIR=$(cd $(dirname "$0") && pwd)

cd $SCRIPTDIR

docker-compose up -d
