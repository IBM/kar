############################################################
# Install KAR into kind cluster on macos
#
# Prerequisites: Docker and Kind should be installed already
# versions: Docker version 19.03.8, build afacb8b
#           kind v0.9.0 go1.14.2 darwin/amd64
#############################################################

#!/bin/sh
set -e

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/.."

echo "Running KAR setup ...."
cd $ROOTDIR
$SCRIPTDIR/kind-start.sh
make dockerDev
$SCRIPTDIR/kar-k8s-deploy.sh -dev

echo "Building kar CLI"
make cli

echo "Setting up namespace and environment"
$SCRIPTDIR/kar-k8s-namespace-enable.sh default

source $SCRIPTDIR/kar-env-local.sh

echo "KAR Setup Complete!"
