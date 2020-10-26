kamel run --name=source \
          src/main/java/org/apache/camel/kar/kamel/kafka/InputProcessor.java \
          -p camel.component.kafka.brokers=${KAR_KAFKA_CLUSTER_IP}:9092 \
          input.yaml --dev
