# Trees

This example demonstrates how to activate a large number of actors using a
binary tree. Three implementations are provided.
* Actor `Sync` implements a sequential construction. For each node, the left
  subtree is constructed first, followed by the right subtree.
* Actor `Async` implements a parallel construction. For each node, the left
  subtree and right subtree constructions are started concurrently. The leaf
  actors report to the root actor that keeps a count of expected leaf nodes.
  Since the root actor is waiting inside the `test` invocation while the leaf
  nodes invoke `decr` on it, the `decr` invocations all reuse the session ID
  from the `test` invocation.
* Actor `Par` implements a parallel construction. For each node, the left
  subtree and right subtree constructions are started concurrently. The node
  then waits for the completion of these two subtrees before returning to the
  parent node. Since many concurrent HTTP connections are required to implement
  the waiting, this construction style requires HTTP/2 at scale.

Setup with:
```shell
npm install --prod
```
Run actor runtime with:
```shell
kar run -h2c -actor_collector_interval 120s -v info -app tree -actors Sync,Async,Par -- node server.js
```
Run synchronous tree example with:
```shell
kar run -v info -app tree -- node test-sync.js 10
```
Run asynchronous tree example with:
```shell
kar run -v info -app tree -- node test-async.js 10
```
Run parallel tree example with:
```shell
kar run -v info -app tree -- node test-par.js 10
```
