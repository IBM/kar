SCRIPTDIR=../../scripts

kamel run --name=source \
          $SCRIPTDIR/kamel/CloudEventProcessor.java \
          -p camel.component.kafka.brokers=${KAR_KAFKA_CLUSTER_IP}:9092 \
          input.yaml --dev
