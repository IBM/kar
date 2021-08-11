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

# This script enables the current IBM Code Engine project for
# running KAR applications by creating a secret that
# enables the use of "Databases for Redis" and "Event Streams"
# services on the IBM Cloud.
#
# The script assumes all resources already exist and have
# service-keys created with the required permissions.
#
# The script requires the name of the service-keys as an argument

if [ $# -lt 1 ];
then
   echo "Usage: kar-code-engine-project-enable.sh <service-key>"
   exit 1
fi

set -eu

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/.."

SERVICE_KEY=$1

echo "Extracting credentials from service key"
. ${SCRIPTDIR}/kar-env-ibmcloud.sh $SERVICE_KEY

echo "Creating runtime-config secret in code-engine project"
ibmcloud code-engine secret create --name kar.ibm.com.runtime-config \
     --from-literal REDIS_ENABLE_TLS=$REDIS_ENABLE_TLS \
     --from-literal REDIS_CA=$REDIS_CA \
     --from-literal REDIS_HOST=$REDIS_HOST \
     --from-literal REDIS_PORT=$REDIS_PORT \
     --from-literal REDIS_PASSWORD=$REDIS_PASSWORD \
     --from-literal REDIS_USER=$REDIS_USER \
     --from-literal KAFKA_VERSION=$KAFKA_VERSION \
     --from-literal KAFKA_ENABLE_TLS=$KAFKA_ENABLE_TLS \
     --from-literal KAFKA_BROKERS=$KAFKA_BROKERS \
     --from-literal KAFKA_PASSWORD=$KAFKA_PASSWORD
