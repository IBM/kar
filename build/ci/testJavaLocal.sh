#!/bin/bash
set -eu

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/../.."

echo "Executing Java tests"
cd $ROOTDIR/sdk/java
mvn clean test