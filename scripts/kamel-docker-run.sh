#!/bin/bash

# Script to run a kamel integration on docker

docker run --network kar-bus --env KAFKA_BROKERS=kafka:9092 "$@"
