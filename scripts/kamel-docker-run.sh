#!/bin/bash

# run integration in a local container

set -e

SCRIPTDIR=$(cd $(dirname "$0") && pwd)

image=

while [ $# -gt 0 ]; do
  case "$1" in
    --image|-i)
      shift
      image="$1"
      shift
      ;;
    *)
      array+=("$1")
      shift
      ;;
  esac
done

set -- "${array[@]}"

kamel local run --containerize \
  --image ${image} \
  --network kar-bus \
  "${SCRIPTDIR}/kamel/CloudEventProcessor.java" \
  "$@"
