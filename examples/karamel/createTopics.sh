#!/bin/bash

echo "*** Create topics for end-to-end stock processing."
echo "*** For now Karamel does not support the automatic creation of topics."
echo "*** Topic:"
echo "***   InputStockEvent"
echo "***   OutputStockEvent"
kubectl exec kar-kafka-0 -n kar-system -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server :9092 --create --topic InputStockEvent
kubectl exec kar-kafka-0 -n kar-system -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server :9092 --create --topic OutputStockEvent
