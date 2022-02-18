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

# Source this script to setup your shell environment for local app
# invocation using a "Databases for Redis" and "Event Streams"
# services on the IBM Cloud. The script requires the name of the
# service credential keys (same name for both Redis and Event Streams).
# The Event Streams key must permit manager access.
# The script expects the user is already logged in.
#
# Usage . kar-env-ibmcloud.sh <service-key>
#

if [ $# -lt 1 ]; 
then 
   echo ". kar-env-ibmcloud.sh <service-key>"
   return 1
fi

# fetch keys, must already be logged in
KEY=`ibmcloud resource service-key $1 --output json`

# extract redis key
REDIS_KEY=`echo $KEY | jq '.[] | select(.source_crn|test("redis"))'`

# extract kafka key
KAFKA_KEY=`echo $KEY | jq '.[] | select(.source_crn|test("messagehub"))'`

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

# setup redis env variables
export REDIS_ENABLE_TLS=true
export REDIS_HOST=`echo $REDIS_KEY | jq -r .credentials.connection.rediss.hosts[0].hostname`
export REDIS_PORT=`echo $REDIS_KEY | jq -r .credentials.connection.rediss.hosts[0].port`
export REDIS_PASSWORD=`echo $REDIS_KEY | jq -r .credentials.connection.rediss.authentication.password`
export REDIS_USER=`echo $REDIS_KEY | jq -r .credentials.connection.rediss.authentication.username`
export REDIS_CA=`echo $REDIS_KEY | jq -r .credentials.connection.rediss.certificate.certificate_base64`

# setup kafka env variables
export KAFKA_VERSION=2.3.0
export KAFKA_ENABLE_TLS=true
export KAFKA_BROKERS=`echo $KAFKA_KEY | jq -r .credentials.kafka_brokers_sasl[0]` # TODO: use all brokers
export KAFKA_PASSWORD=`echo $KAFKA_KEY | jq -r .credentials.password`
