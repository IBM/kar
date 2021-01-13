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
