SCRIPTDIR=../../scripts

kamel run --name=sink \
          $SCRIPTDIR/kamel/CloudEventProcessor.java \
          -p camel.component.kafka.brokers=${KAR_KAFKA_CLUSTER_IP}:9092 \
          -p camel.component.slack.webhookUrl=${SLACK_KAR_OUTPUT_WEBHOOK} \
          output.yaml --dev
