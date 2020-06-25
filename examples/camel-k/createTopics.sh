#!/bin/bash

echo "*** Create topics for Camel-based console application. ***"
echo "*** Topics:"
echo "***   1. CamelKEvent"
kubectl exec kar-kafka-0 -n kar-system -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server :9092 --create --topic CamelKEvent
