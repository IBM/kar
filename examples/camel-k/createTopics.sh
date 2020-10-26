#!/bin/bash

echo "*** Create topics for end-to-end stock processing Camel application. ***"
echo "*** Topic:"
echo "***   InputStockEvent"
echo "***   OutputStockEvent"

kafka-topics --bootstrap-server localhost:31093 --create --topic InputStockEvent
kafka-topics --bootstrap-server localhost:31093 --create --topic OutputStockEvent
