#!/bin/bash

# Script to teardown the KAR runtime launched using docker-compose

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
cd $SCRIPTDIR

docker-compose down

docker network rm kar-bus
