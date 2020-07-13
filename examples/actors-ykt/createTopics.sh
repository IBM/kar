#!/bin/bash

echo "*** Create topics used by IBM Research Site Simulation ***"
echo "*** Topics:"
echo "***   1. siteReport"
echo "***   2. outputReport"
kubectl exec kar-kafka-0 -n kar-system -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server :9092 --create --topic siteReport
kubectl exec kar-kafka-0 -n kar-system -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server :9092 --create --topic outputReport

