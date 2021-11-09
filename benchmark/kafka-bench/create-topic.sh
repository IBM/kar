# Topic names: simple-topic, return-topic

rf=$1

echo "creating topic with replication-factor $rf"

kubectl exec kar-kafka-0 -n kar-system -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server :9092 --create --replication-factor $rf --topic return-topic
kubectl exec kar-kafka-0 -n kar-system -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server :9092 --create --replication-factor $rf --topic simple-topic

