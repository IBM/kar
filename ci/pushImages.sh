#!/bin/bash

#
# Copyright IBM Corporation 2020,2022
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

cd $ROOTDIR

docker login -u "${QUAY_USERNAME}" -p "${QUAY_PASSWORD}" quay.io

if [ ${TRAVIS_BRANCH} == "main" ]; then
    # push `latest` tag images
    KAR_VERSION=$(git rev-parse --short ${TRAVIS_COMMIT}) DOCKER_REGISTRY=quay.io DOCKER_NAMESPACE=ibm DOCKER_IMAGE_TAG=latest make docker
else
    # push tagged images
    KAR_VERSION="${TRAVIS_BRANCH:1}" DOCKER_REGISTRY=quay.io DOCKER_NAMESPACE=ibm DOCKER_IMAGE_TAG="${TRAVIS_BRANCH:1}" make docker
fi
