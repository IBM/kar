# Example

```sh
npm install

export KAFKA_BROKERS=... # kafka brokers as a comma separated list
export KAFKA_USER=... # optional, user name
export KAFKA_PASSWORD=... # optional, password or api key
export KAFKA_TLS=... # optional, set to true to enable TLS
export KAFKA_VERSION=... # optional

export REDIS_ADDRESS=... # address of redis server
export REDIS_PASSWORD=... # optional, password of redis server
export REDIS_TLS=... # optional, set to true to enable TLS

# run server
kar -app myApp -service myService node server.js &

# run client
kar -app myApp -service myClient node client.js
```
