# Source this script to setup your shell environment for local app
# invocation using a KAR runtime deployed in a kind cluster.
#
# Usage . kar-kind-env.sh
#

export KAFKA_BROKERS=localhost:31093
export KAFKA_VERSION=2.4.0
export REDIS_HOST=localhost
export REDIS_PORT=31379
export REDIS_PASSWORD=passw0rd
