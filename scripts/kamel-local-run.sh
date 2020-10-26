#!/bin/bash

# Script to run a kamel integration locally

set -e

SCRIPTDIR=$(cd $(dirname "$0") && pwd)

# extract workspace flag

workspace=workspace

while [ $# -gt 0 ]; do
  case "$1" in
    --workspace)
      shift
      workspace="$1"
      shift
      ;;
    *)
      echo "Error: unexpected argument: $1"
      exit 1
      ;;
  esac
done

# run integration

cd "${workspace}"

function assemble { printf "file:"; while [ $# -gt 1 ]; do printf "%s%s" "$1" ",file:"; shift; done; printf "%s" "$1"; }

CAMEL_K_CONF_D=properties \
CAMEL_K_ROUTES=$(assemble $(find src -type f)) \
java -cp "dependencies/*" org.apache.camel.k.main.Application
