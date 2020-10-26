kamel run --name=sink \
          src/main/java/org/apache/camel/kar/kamel/kafka/OutputProcessor.java \
          -p camel.component.kafka.brokers=${KAR_KAFKA_CLUSTER_IP}:9092 \
          -p camel.component.slack.webhookUrl=${SLACK_KAR_OUTPUT_WEBHOOK} \
          output.yaml --dev
