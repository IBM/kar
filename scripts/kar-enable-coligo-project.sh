#!/bin/bash

# This script enables the current coligo project for
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
   echo "Usage: kar-enable-coligo-project.sh <service-key>"
   exit 1
fi

set -eu

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/.."

echo "Extracting credentials from service key"
. ${SCRIPTDIR}/kar-cloud-env.sh $1

echo "Creating secret in coligo project"
ibmcloud coligo secret create --name kar.ibm.com.runtime-config \
     --from-literal REDIS_ENABLE_TLS=$REDIS_ENABLE_TLS \
     --from-literal REDIS_HOST=$REDIS_HOST \
     --from-literal REDIS_PORT=$REDIS_PORT \
     --from-literal REDIS_PASSWORD=$REDIS_PASSWORD \
     --from-literal KAFKA_VERSION=$KAFKA_VERSION \
     --from-literal KAFKA_ENABLE_TLS=$KAFKA_ENABLE_TLS \
     --from-literal KAFKA_BROKERS=$KAFKA_BROKERS \
     --from-literal KAFKA_PASSWORD=$KAFKA_PASSWORD
