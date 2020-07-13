SCRIPTDIR=$(cd $(dirname "$0") && pwd)
CODEDIR="$SCRIPTDIR/../../kar-kamel"

kamel run --name=output \
          $CODEDIR/src/main/java/org/apache/camel/kar/kamel/kafka/OutputConfiguration.java \
          -p camel.component.kafka.brokers=${KAR_KAFKA_CLUSTER_IP}:9092 \
          -p camel.component.slack.webhookUrl=${SLACK_KAR_OUTPUT_WEBHOOK} \
          $SCRIPTDIR/outputSlack.yaml --dev
