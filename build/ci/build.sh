#!/bin/bash

set -eu

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/../.."
cd $ROOTDIR

DOCKER_IMAGE_PREFIX= DOCKER_IMAGE_TAG=dev make install kindPush
