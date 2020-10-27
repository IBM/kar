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
  "${SCRIPTDIR}/kamel/CloudEventProcessor.java" \
  "$@"

# create folders

mkdir -p "${workspace}/properties"
mkdir -p "${workspace}/src"

# copy source files

cp "$@" "${workspace}/src"
cp "${SCRIPTDIR}/kamel/CloudEventProcessor.java" "${workspace}/src"

# copy Dockerfile

cp "${SCRIPTDIR}/kamel/Dockerfile" "${workspace}"
