SCRIPTDIR=../../scripts

set -e

SECRET=$(kubectl get -n kar-system secret/kar.ibm.com.runtime-config -o json)
BROKERS=$(echo $SECRET | jq -r .data.kafka_brokers | base64 -D)

kamel run \
  -p camel.component.kafka.brokers=$BROKERS \
  $SCRIPTDIR/kamel/CloudEventProcessor.java \
  --name=sink output.yaml -p camel.component.slack.webhookUrl=${SLACK_KAR_OUTPUT_WEBHOOK} --dev
