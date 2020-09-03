############################################################
# Install KAR into kind cluster on macos
#
# Prerequisites: Docker and Kind should be installed already
# versions: Docker version 19.03.8, build afacb8b
#           kind v0.8.1 go1.14.2 darwin/amd64
#############################################################

#!/bin/sh
set -e

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/.."

echo "Running KAR setup ...."
cd $ROOTDIR
$SCRIPTDIR/start-kind.sh
make dockerDev
$SCRIPTDIR/kar-deploy.sh -dev

echo "Building kar CLI"
make install

echo "Setting up namespace and environment"
$SCRIPTDIR/kar-enable-namespace.sh default

source $SCRIPTDIR/kar-kind-env.sh

echo "KAR Setup Complete!"