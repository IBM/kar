#!/bin/bash

# Script to build a kamel integration locally

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
    *)
      array+=("$1")
      shift
      ;;
  esac
done

# run kamel inspect

kamel inspect --all-dependencies \
  --additional-dependencies camel-k:runtime-main,github:cloudevents/sdk-java/f42020333a8ecfa6353fec26e4b9d6eceb97e626 \
  $SCRIPTDIR/kamel/org/apache/camel/kar/kamel/kafka/*.java \
  "$@"

# create property file if it does not exist yet

mkdir -p $workspace/properties

if [ ! -f $workspace/properties/integration.properties ]; then
  echo camel.component.kafka.brokers=$KAFKA_BROKERS > $workspace/properties/integration.properties
fi

# create run script

function assemble { local f=$1; shift; printf %s "file:$f" "${@/#/,file:}"; }

echo cd $(pwd) > $workspace/run.sh
echo CAMEL_K_ROUTES=$(assemble $SCRIPTDIR/kamel/org/apache/camel/kar/kamel/kafka/*.java "${array[@]}") \
CAMEL_K_CONF_D=$workspace/properties \
java -cp \"$workspace/dependencies/*\" org.apache.camel.k.main.Application >> $workspace/run.sh

chmod u+x $workspace/run.sh
