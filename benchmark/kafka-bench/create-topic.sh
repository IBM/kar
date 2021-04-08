# Topic names: simple-topic, return-topic
kubectl exec kar-kafka-0 -n kar-system -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server :9092 --create --topic return-topic

