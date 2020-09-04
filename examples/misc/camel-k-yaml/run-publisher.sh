kamel run --name=publisher \
          -d github:cloudevents/sdk-java/f42020333a8ecfa6353fec26e4b9d6eceb97e626 \
          -d camel:camel-gson \
          -d camel:camel-kafka \
          --property camel.component.kafka.brokers=${KAR_KAFKA_CLUSTER_IP}:9092 \
          publisher.yaml src/main/java/org/apache/camel/yaml/kafka/PublisherConfiguration.java --dev
