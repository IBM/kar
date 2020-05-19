#!/bin/bash

# Script to stop kafka and redis when launched using docker-compose

SCRIPTDIR=$(cd $(dirname "$0") && pwd)

cd $SCRIPTDIR

docker-compose down
