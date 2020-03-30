# Example

```sh
npm install

export KAFKA_BROKERS=... # kafka brokers as a comma separated list
export KAFKA_USERNAME=... # optional, user name
export KAFKA_PASSWORD=... # optional, password or api key
export KAFKA_ENABLE_TLS=... # optional, set to true to enable TLS
export KAFKA_VERSION=... # optional

export REDIS_HOST=... # redis host
export REDIS_PORT=... # optional, redis port
export REDIS_PASSWORD=... # optional, password of redis server
export REDIS_ENABLE_TLS=... # optional, set to true to enable TLS

# run server
kar -app myApp -service myService -actors Foo node server.js &

# run a client
kar -app myApp -service myClient1 node client.js

kar -app myApp -service myClient2 node actor.js

kar -app myApp -service myClient3 node actor-reminder.js

```
