#!/bin/bash

echo "*** Create topics for end-to-end stock processing Camel application. ***"
echo "*** Topic:"
echo "***   StockEvent"
kubectl exec kar-kafka-0 -n kar-system -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server :9092 --create --topic StockEvent
