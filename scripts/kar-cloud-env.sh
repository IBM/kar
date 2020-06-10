# Source this script to setup your shell environment for local app
# invocation using a "Databases for Redis" and "Event Streams"
# services on the IBM Cloud. The script requires the name of the
# service credential keys (same name for both Redis and Event Streams).
# The Event Streams key must permit manager access.
# The script expects the user is already logged in.
#
# Usage . kar-cloud-env.sh <service-key>
#

if [ $# -lt 1 ]; 
then 
   echo ". kar-cloud-env.sh <service-key>"
   return 1
fi

# fetch keys, must already be logged in
KEY=`ibmcloud resource service-key $1 --output json`

# extract redis key
REDIS_KEY=`echo $KEY | jq '.[] | select(.source_crn|test("redis"))'`

# extract kafka key
KAFKA_KEY=`echo $KEY | jq '.[] | select(.source_crn|test("messagehub"))'`

# setup redis env variables
export REDIS_ENABLE_TLS=true
export REDIS_HOST=`echo $REDIS_KEY | jq -r .credentials.connection.rediss.hosts[0].hostname`
export REDIS_PORT=`echo $REDIS_KEY | jq -r .credentials.connection.rediss.hosts[0].port`
export REDIS_PASSWORD=`echo $REDIS_KEY | jq -r .credentials.connection.rediss.authentication.password`

# setup kafka env variables
export KAFKA_VERSION=2.2.0
export KAFKA_ENABLE_TLS=true
export KAFKA_BROKERS=`echo $KAFKA_KEY | jq -r .credentials.kafka_brokers_sasl[0]` # TODO: use all brokers
export KAFKA_PASSWORD=`echo $KAFKA_KEY | jq -r .credentials.password`
