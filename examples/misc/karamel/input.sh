export KARAMEL_ROOT=~/K/karamel
export KARAMEL_HTTP_ADDRESS=http://financialmodelingprep.com/api/v3/quote-short/AAPL?apikey=demo

kar run -v info -app stocks -runtime_port 3503 -app_port 8083 -- \
    karamel -camel_components http -kafka_topics InputStockEvent -pub -workspace $PWD/input
