#!/bin/bash

set -eu

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/.."

$ROOTDIR/scripts/kar-deploy.sh -dev

$ROOTDIR/scripts/kar-enable-namespace.sh default
