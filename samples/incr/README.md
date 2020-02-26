# Example

```sh
npm install

export KAFKA_BROKERS=... # kafka brokers as a comma separated list
export KAFKA_USER=... # user name or "token" if using an api key
export KAFKA_PASSWORD=... # password or api key
export KAFKA_TLS=... # true or false

# run server
kar -app myApp -service myService -launch node server.js &

# run client
kar -app myApp -service myClient -launch node client.js
```
