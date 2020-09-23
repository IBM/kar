#!/bin/bash

echo "*** Create topics used by Stock Pricing Application ***"
echo "*** Topics:"
echo "***   1. historical-prices"
echo "***   2. buy-stock"
kubectl exec kar-kafka-0 -n kar-system -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server :9092 --create --topic historical-prices
kubectl exec kar-kafka-0 -n kar-system -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server :9092 --create --topic buy-stock

