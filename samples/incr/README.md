# Example

```sh
npm install

export KAFKA_BROKERS=... # kafka brokers as a comma separated list
export KAFKA_USER=... # user name or "token" if using an api key
export KAFKA_PASSWORD=... # password or api key

# run server
kar -app myApp -service myService -tls node server.js &

# run client
kar -app myApp -service myClient -tls node client.js
```
