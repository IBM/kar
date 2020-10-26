#!/bin/bash

# Script to build a kamel integration locally

set -e

SCRIPTDIR=$(cd $(dirname "$0") && pwd)

# extract workspace flag

array=()
workspace=workspace

while [ $# -gt 0 ]; do
  case "$1" in
    --workspace)
      shift
      workspace="$1"
      shift
      ;;
    -*)
      echo "Error: unexpected argument: $1"
      exit 1
      ;;
    *)
      array+=("$1")
      shift
      ;;
  esac
done

set -- "${array[@]}"

# run kamel inspect

kamel inspect --all-dependencies \
  --workspace "${workspace}" \
  --additional-dependencies camel-k:runtime-main,github:cloudevents/sdk-java/f42020333a8ecfa6353fec26e4b9d6eceb97e626 \
  "${SCRIPTDIR}/kamel/org/apache/camel/kar/kamel/kafka/InputProcessor.java" \
  "${SCRIPTDIR}/kamel/org/apache/camel/kar/kamel/kafka/OutputProcessor.java" \
  "$@"

# create kafka.properties file

mkdir -p "${workspace}/properties"

echo camel.component.kafka.brokers=$KAFKA_BROKERS > "${workspace}/properties/kafka.properties"
if [ -z $KAFKA_BROKERS ]; then
  echo "Warning: please set property camel.component.kafka.brokers in properties/kafka.properties"
fi

# copy source files

mkdir -p "${workspace}/src"
cp "$@" "${workspace}/src"
cp "${SCRIPTDIR}/kamel/org/apache/camel/kar/kamel/kafka/InputProcessor.java" "${workspace}/src"
cp "${SCRIPTDIR}/kamel/org/apache/camel/kar/kamel/kafka/OutputProcessor.java" "${workspace}/src"
