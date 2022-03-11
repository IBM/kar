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

set -e

# Run the servers:
( kar run -h2c -app unit-test -app_port 8081 -actors TestActor -service sdk-test python actor_server.py ) &
( kar run -h2c -app unit-test -app_port 8082 -service sdk-test-services python service_server.py ) &

# Wait for server to start:
sleep 2

# Run the client:
kar run -h2c -app unit-test pytest
