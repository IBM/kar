#!/bin/bash

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

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/.."

# Run local version of actors-python.
echo "*** Testing examples/actors-python ***"

# Move into the example directory:
cd $ROOTDIR/examples/actors-python

# Launch test:
sh launch.sh

# Run local version of the python unit tests.
echo "*** Testing examples/python-unit-tests ***"

# Move into the example directory:
cd $ROOTDIR/examples/python-unit-tests

# Launch test:
sh launch.sh
