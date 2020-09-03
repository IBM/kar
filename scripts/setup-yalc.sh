#!/bin/bash

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/.."

cd $ROOTDIR/sdk-js
yalc publish

EXAMPLES=$(find $ROOTDIR/examples -name .yalc -print0 -maxdepth 2 | xargs -0 -n1 dirname)

for e in $EXAMPLES
do
    cd $e
    yalc update
done
