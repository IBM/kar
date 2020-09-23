# Hello World Example

This example demonstrates how to use KAR's REST API directly. It consists of an
[HTTP server](server.js) implemented using `express` and an [HTTP
client](client.js) implemented using `node-fetch` and `fetch-retry`.

Detailed instructions for running the example are contained in [Getting
Started](../../docs/getting-started.md).

## Server Code

The server code uses [express](https://www.npmjs.com/package/express) to standup
a REST server with two routes:
* The `helloText` route handles a `POST` with a request body with mime type
  `text/plain`. 
* The `helloJson` route handles a `POST` with a request body with mime type
  `application/json`.

The parsing of the request bodies is handled using builtin express parsers.

The server listens on the port specified by environment variable `KAR_APP_PORT`
that is expected to be set by the KAR application launcher.

As a simple security measure, the server only listens for requests coming from
`127.0.0.1` since all requests are expected to come from a KAR sidecar process
running on the same host (or pod).

## Client Code

The client code use [node-fetch](https://www.npmjs.com/package/node-fetch) to
invoke the two routes in turn.

The client makes requests via its KAR sidecar process using port
`KAR_RUNTIME_PORT` that is expected to be set by the KAR application launcher.

The `url` function maps a service and route to the matching API of the KAR
sidecar.

The `fetch` function call is wrapped using
[fetch-retry](https://www.npmjs.com/package/fetch-retry) to retry requests up to
10 times over 10 seconds. These retries are intended to mask transient
communication errors between the client process and the KAR process that may
occur at startup time.
