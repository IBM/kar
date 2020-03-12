#!/bin/bash

set -ex

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/../../"
cd $ROOTDIR

DOCKER_IMAGE_PREFIX=kar-travis make docker
