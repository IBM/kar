#!/bin/bash

set -eux

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/../"

IMAGE_TAG=$1

if [[ ! -z ${DOCKER_PASSWORD} ]]; then
  docker login -u iamapikey -p "${DOCKER_APIKEY}" us.icr.io
fi

cd $ROOTDIR

make dockerPush

# if doing nightly also push a tag with the hash commit
if [ ${IMAGE_TAG} == "nightly" ]; then
  SHORT_COMMIT=`git rev-parse --short HEAD`
  IMAGE_TAG=dev-${SHORT_COMMIT} make dockerPush
fi
