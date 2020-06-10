# Source this script to setup your shell environment for local app
# invocation using a "Databases for Redis" and "Event Streams"
# services on the IBM Cloud. 
#
# Usage . kar-cloud-env.sh
#

if [ $# -lt 1 ]; 
then 
   echo ". kar-cloud-env.sh <service-key>"
   return 1
fi

KEY=`ibmcloud resource service-key $1 --output json`
REDIS_KEY=`echo $KEY | jq '.[] | select(.source_crn|test("redis"))'`
KAFKA_KEY=`echo $KEY | jq '.[] | select(.source_crn|test("messagehub"))'`

export REDIS_ENABLE_TLS=true
export REDIS_HOST=`echo $REDIS_KEY | jq -r .credentials.connection.rediss.hosts[0].hostname`
export REDIS_PORT=`echo $REDIS_KEY | jq -r .credentials.connection.rediss.hosts[0].port`
export REDIS_PASSWORD=`echo $REDIS_KEY | jq -r .credentials.connection.rediss.authentication.password`

export KAFKA_VERSION=2.2.0
export KAFKA_ENABLE_TLS=true
export KAFKA_BROKERS=`echo $KAFKA_KEY | jq -r .credentials.kafka_brokers_sasl[0]`
export KAFKA_PASSWORD=`echo $KAFKA_KEY | jq -r .credentials.password`
