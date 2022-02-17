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

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/.."

KAR_PYTHON_SDK=${DOCKER_IMAGE_PREFIX}kar-sdk-python-v1:${DOCKER_IMAGE_TAG}
KAR_EXAMPLE_ACTORS_PYTHON_CONTAINERIZED=


# Run containerized version of actors-python. Inside the container
# the example runs locally.
echo "*** Testing examples/actors-python ***"

# Move into the example directory:
cd examples/actors-python

# Build the example image for the containerized example:
docker build -f Dockerfile.containerized --build-arg PYTHON_RUNTIME=$KAR_PYTHON_SDK -t $KAR_EXAMPLE_ACTORS_PYTHON_CONTAINERIZED .

# Run the example inside a docker container:
docker run --network kar-bus --add-host=host.docker.internal:host-gateway $KAR_EXAMPLE_ACTORS_PYTHON_CONTAINERIZED
