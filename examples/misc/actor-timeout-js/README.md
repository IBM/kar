# Actor Timeout

This example demonstrates how a deadlock when trying to acquire an actor
instance is reported as a timeout error.
```
npm install
kar run -app actor_timeout -actors Test -actor_timeout 10s -- node timeout.js
```
