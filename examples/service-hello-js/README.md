# Hello World Example

This example demonstrates how to use KAR's REST API directly. It consists of an
[HTTP server](server.js) implemented using `express` and an [HTTP
client](client.js) implemented using `node-fetch` and `fetch-retry`.

## Building the Code

You will need Node 12 (LTS) and NPM 6.12+ installed.
Build the code by doing `npm install --prod`.

## Run the Server without KAR and interact via curl:

In one window:
```shell
(%) KAR_APP_PORT=8080 node server.js
```

In a second window, invoke routes using curl
```shell
(%) curl -s -X POST -H "Content-Type: text/plain" http://localhost:8080/helloText -d 'Gandalf the Grey'
Hello Gandalf the Grey
```
```shell
(%) curl -s -X POST -H "Content-Type: application/json" http://localhost:8080/helloJson -d '{"name": "Alan Turing"}'
{"greetings":"Hello Alan Turing"}
```

## Run using KAR

In one window:
```shell
(%) kar run -app hello-js -service greeter node server.js
```

In a second window:
```shell
(%) kar run -app hello-js node client.js
```

You should see output like shown below in both windows:
```
2020/04/02 17:41:23 [STDOUT] Hello John Doe!
2020/04/02 17:41:23 [STDOUT] Hello John Doe!
```
The client process will exit, while the server remains running. You
can send another request, or exit the server with a Control-C.

You can use the `kar` cli to invoke a route directly (the content type for request bodies defaults to application/json).
```shell
(%) kar rest -app hello-js post greeter helloJson '{"name": "Alan Turing"}'
2020/10/06 10:04:27.014025 [STDOUT] {"greetings":"Hello Alan Turing!"}
```

Or invoke the `text/plain` route with an explicit content type:
```shell
(%) kar rest -app hello-js -content_type text/plain post greeter helloText 'Gandalf the Grey'
2020/10/06 09:48:29.644326 [STDOUT] Hello Gandalf the Grey
```

If the service endpoint being invoked requires more sophisticated
headers or other features not supported by the `kar rest` command, it
is still possible to use curl. However, the curl command is now using
KAR's REST API to make the service call via a `kar` sidecar.

```shell
(%) kar run -runtime_port 32123 -app hello-js curl -s -X POST -H "Content-Type: text/plain" http://localhost:32123/kar/v1/service/greeter/call/helloText -d 'Gandalf the Grey'
2020/10/06 09:49:45.300122 [STDOUT] Hello Gandalf the Grey
```

## Looking Inside the Server Code

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

## Looking Inside the Client Code

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
