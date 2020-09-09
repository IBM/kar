export KARAMEL_ROOT=~/K/karamel
export KARAMEL_SLACK_CHANNEL=kar-output

kar run -v info -app stocks -runtime_port 3502 -app_port 8082 -- \
  karamel -camel_components slack -kafka_topics OutputStockEvent -sub -workspace $PWD/output