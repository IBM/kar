#!/bin/bash

echo "*** Creating test topics ***"
echo "*** Topics:"
echo "***   1. topic1"
echo "***   2. topic2"
kubectl exec kar-kafka-0 -n kar-system -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server :9092 --create --topic topic1
kubectl exec kar-kafka-0 -n kar-system -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server :9092 --create --topic topic2
