#!/bin/bash

#
# Copyright IBM Corporation 2020,2023
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

# Script to automate installation of KAR runtime into a Kubernetes cluster

set -e

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/.."

if command -v kar &> /dev/null; then
    version=$(kar version)
    case "$version" in
        unofficial)
            helmargs="$helmargs --set-string kar.injector.imageName=registry:5000/kar/kar-injector --set-string kar.injector.sidecarImageName=registry:5000/kar/kar-sidecar"
            kartag="latest";;
        *)
            helmargs="$helmargs --set-string kar.injector.imageName=quay.io/ibm/kar-injector --set-string kar.injector.sidecarImageName=quay.io/ibm/kar-sidecar"
            kartag="$version";;
    esac
else
    echo "No kar cli in path. Unable to infer desired KAR version; must specify explicitly"
    kartag=""
fi

help=""
args=""
parse=true
injectorOnly=""
openshift=""
agent=""
while [ -n "$1" ]; do
    if [ -z "$parse" ]; then
        args="$args '$1'"
        shift
        continue
    fi
    case "$1" in
	-a|-agent|--agent) agent="1"; break;;
        -h|-help|--help) help="1"; break;;
        -os|-openshift) openshift="1";;
        -m|-managed|--managed)
            shift;
            serviceKey="$1"
            . $SCRIPTDIR/kar-env-ibmcloud.sh $serviceKey
            helmargs="$helmargs --set kafka.internal=false --set redis.internal=false"
            helmargs="$helmargs --set-string kafka.externalConfig.enabletls=$KAFKA_ENABLE_TLS"
            helmargs="$helmargs --set-string kafka.externalConfig.version=$KAFKA_VERSION"
            helmargs="$helmargs --set-string kafka.externalConfig.brokers=$KAFKA_BROKERS"
            helmargs="$helmargs --set-string kafka.externalConfig.password=$KAFKA_PASSWORD"
            helmargs="$helmargs --set-string redis.externalConfig.enabletls=$REDIS_ENABLE_TLS"
            helmargs="$helmargs --set-string redis.externalConfig.host=$REDIS_HOST"
            helmargs="$helmargs --set-string redis.externalConfig.port=$REDIS_PORT"
            helmargs="$helmargs --set-string redis.externalConfig.password=$REDIS_PASSWORD"
            helmargs="$helmargs --set-string redis.externalConfig.user=$REDIS_USER"
            helmargs="$helmargs --set-string redis.externalConfig.ca=$REDIS_CA"
            injectorOnly="true"
            ;;
        -f|-file|--file) shift; helmargs="$helmargs -f $1";;
        -s|-set|--set) shift; helmargs="$helmargs --set $1";;
        -ss|-set-string|--set-string) shift; helmargs="$helmargs --set-string $1";;
        -r|-release|--release)
            shift;
            helmargs="$helmargs --set-string kar.injector.imageName=quay.io/ibm/kar-injector --set-string kar.injector.sidecarImageName=quay.io/ibm/kar-sidecar"
            kartag="$1";;
        --) parse=;;
        *) args="$args '$1'";;
    esac
    shift
done

if [ -n "$help" ]; then
    cat << EOF
Usage: kar-deploy.sh [options]
where [options] includes:
    -a -agent                 embeds prometheus exporter into kafka to expose metrics
    -f -file <config.yaml>    pass -f <config.yaml> to helm install kar
    -m -managed <service-key> use managed EventStreams and Redis accessed via service-key
    -s -set key=value         pass --set key=value to helm install kar
    -r -release <version>     deploy the specified release
    -os -openshift            deploy to OpenShift
EOF
    exit 0
fi

if [ -n "$kartag" ]; then
    helmargs="$helmargs --set-string kar.version=$kartag "
fi

cd $ROOTDIR

kubectl create namespace kar-system 2>/dev/null || true

if [ -n "$agent" ]; then
    echo "Deploying prometheus metrics exporter configMap"
    kubectl create -f $SCRIPTDIR/kafka-jmx-exporter-cm.yaml
fi

if [ -n "$openshift" ]; then
    oc create sa sa-with-anyuid -n kar-system
    oc adm policy add-scc-to-user anyuid -z sa-with-anyuid -n kar-system
    helmargs="$helmargs --set-string global.openshift=true"
fi

echo "Deploying KAR runtime; this may take several minutes"
helm install kar scripts/helm/kar --wait -n kar-system $helmargs

# Enable default namespace
echo "Enabling default namespace for KAR applications"
$SCRIPTDIR/kar-k8s-namespace-enable.sh default
