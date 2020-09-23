# Source this script to setup your shell environment for local app
# invocation using a KAR runtime deployed in a kind cluster or
# using docker-compose.
#
# Usage . kar-kind-env.sh
#

export KAFKA_BROKERS=${KAFKA_DEPLOY_HOST:-localhost}:31093
export KAFKA_VERSION=2.4.0
export REDIS_HOST=localhost
export REDIS_PORT=31379
export REDIS_PASSWORD=passw0rd
