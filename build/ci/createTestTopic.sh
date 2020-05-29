#!/bin/bash

set -eu

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/../.."

echo "*** Creating test topics ***"
kubectl exec kar-kafka-0 -n kar-system -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server :9092 --create --topic topic1
kubectl exec kar-kafka-0 -n kar-system -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server :9092 --create --topic topic2
