#
# Copyright IBM Corporation 2020,2021
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

SCRIPTDIR=../../scripts

set -e

namespace=

while [ $# -gt 0 ]; do
  case "$1" in
    --namespace|-n)
      shift
      namespace="$1"
      shift
      ;;
    *)
      shift
      ;;
  esac
done

if [ -z $namespace ]; then
  SECRET=$(kubectl get secret/kar.ibm.com.runtime-config -o json)
else
  SECRET=$(kubectl get -n $namespace secret/kar.ibm.com.runtime-config -o json)
fi

BROKERS=$(echo $SECRET | jq -r .data.kafka_brokers | base64 -D)

KAFKA_BROKERS=$BROKERS kamel run \
  $SCRIPTDIR/kamel/CloudEventProcessor.java \
  "$@"
