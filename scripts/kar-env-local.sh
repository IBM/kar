# Source this script to setup your shell environment to
# connect to a KAR runtime deployed locally in one of two ways:
#  1. on docker-compose using docker-compose-start.sh
#  2. on kind using kind-start.sh
#
# Usage . kar-env-local.sh
#

export KAFKA_BROKERS=${KAFKA_DEPLOY_HOST:-localhost}:31093
export KAFKA_VERSION=2.4.0
export REDIS_HOST=localhost
export REDIS_PORT=31379
export REDIS_PASSWORD=passw0rd
