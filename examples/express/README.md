# Hello World Example

This example demonstrates how to use KAR's REST API directly. It consists of
an [HTTP server](server.js) implemented using `express` and an [HTTP client](client.js) implemented using
`node-fetch` and `fetch-retry`.

Install dependencies with:
```sh
npm install
```
Run server with:
```sh
kar -app express -service greeter -- node server.js
```
Run client with:
```sh
kar -app express -- node client.js
```