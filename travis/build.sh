#!/bin/bash

set -eu

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/.."
cd $ROOTDIR

make install dockerDev
