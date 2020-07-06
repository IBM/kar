kamel run --name=sink \
          -d github:cloudevents/sdk-java/f42020333a8ecfa6353fec26e4b9d6eceb97e626 \
          -d camel:camel-kafka \
          -d camel:camel-slack \
          --property camel.component.kafka.brokers=${KAR_KAFKA_CLUSTER_IP}:9092 \
          --property camel.component.slack.webhookUrl=${SLACK_KAR_OUTPUT_WEBHOOK} \
          -e SLACK_KAR_OUTPUT_WEBHOOK=${SLACK_KAR_OUTPUT_WEBHOOK} \
          output.yaml src/main/java/org/apache/camel/kar/kamel/kafka/OutputConfiguration.java --dev
