#!/bin/bash

#
# Copyright IBM Corporation 2020,2021
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

set -eu

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/.."

BRANCH=$1
IMAGE_TAG=$2
cd $ROOTDIR

docker login -u iamapikey -p "${DOCKER_APIKEY_RIS}" us.icr.io

SHORT_COMMIT=`git rev-parse --short HEAD`

if [ ${BRANCH} == "master" ] && [ ${IMAGE_TAG} == "latest" ]; then
    # push commit hash tagged images
    # disable because can't auto-delete old images in RIS namespace
    # DOCKER_NAMESPACE=research/kar-dev DOCKER_IMAGE_TAG=dev-${SHORT_COMMIT} make dockerBuildAndPush

    # push `latest` tag images
    DOCKER_NAMESPACE=research/kar-dev DOCKER_IMAGE_TAG=latest make dockerBuildAndPush
else
    if [ ${BRANCH} == ${IMAGE_TAG} ]; then
        # A git tag operation, push to kar-prod
        DOCKER_NAMESPACE=research/kar-prod DOCKER_IMAGE_TAG=${IMAGE_TAG} make dockerBuildAndPush
    else
        # A push to some branch; push commit-taged image to kar-stage
        DOCKER_NAMESPACE=research/kar-stage DOCKER_IMAGE_TAG=${BRANCH}-${SHORT_COMMIT} make dockerBuildAndPush
    fi
fi
