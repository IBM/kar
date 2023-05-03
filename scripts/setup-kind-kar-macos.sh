#
# Copyright IBM Corporation 2020,2023
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

############################################################
# Install KAR into kind cluster on macos
#
# Prerequisites: Docker and Kind should be installed already
# versions: Docker version 19.03.8, build afacb8b
#           kind v0.10.0 go1.14.2 darwin/amd64
#############################################################

#!/bin/sh
set -e

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/.."

echo "Running KAR setup ...."
cd $ROOTDIR
$SCRIPTDIR/kind-start.sh
make dockerDev
$SCRIPTDIR/kar-k8s-deploy.sh -dev

echo "Building kar CLI"
make cli

source $SCRIPTDIR/kar-env-local.sh

echo "KAR Setup Complete!"
