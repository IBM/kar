#!/bin/bash

echo "*** Create topics for Camel-based console application. ***"
echo "*** Topics:"
echo "***   1. TestLog"
echo "***   2. HelloEvent"
kubectl exec kar-kafka-0 -n kar-system -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server :9092 --create --replication-factor 1 --partitions 2 --topic TestLog
kubectl exec kar-kafka-0 -n kar-system -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server :9092 --create --topic HelloEvent
