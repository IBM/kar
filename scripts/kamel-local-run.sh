#!/bin/bash

# script to run a kamel integration locally

set -e

SCRIPTDIR=$(cd $(dirname "$0") && pwd)

# run integration locally by invoking kamel local run

kamel local run \
  "${SCRIPTDIR}/kamel/CloudEventProcessor.java" \
  "$@"
