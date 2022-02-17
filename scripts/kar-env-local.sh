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

# Source this script to setup your shell environment to
# connect to a KAR runtime deployed locally in one of two ways:
#  1. on docker-compose using docker-compose-start.sh
#  2. on kind using kind-start.sh
#
# Usage . kar-env-local.sh
#

# Clear any old bindings
unset REDIS_ENABLE_TLS
unset REDIS_HOST
unset REDIS_PORT
unset REDIS_PASSWORD
unset REDIS_USER
unset KAFKA_VERSION
unset KAFKA_ENABLE_TLS
unset KAFKA_BROKERS
unset KAFKA_PASSWORD

export KAFKA_BROKERS=${KAFKA_DEPLOY_HOST:-localhost}:31093
export KAFKA_VERSION=2.8.1
export REDIS_HOST=localhost
export REDIS_PORT=31379
export REDIS_PASSWORD=act0rstate
export REDIS_USER=karmesh
