<!--
# Copyright IBM Corporation 2020,2023
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
-->

# Python SDK for KAR

An implementation of the KAR API in Python.

## Examples and tests

KAR provides an [actor example](https://github.com/IBM/kar/tree/main/examples/service-hello-python) using the KAR API directly from Python.

Examples using KAR's Python SDK:

- [actors-python](https://github.com/IBM/kar/tree/main/examples/actors-python)

The automatic testing system of KAR uses a set of [Python SDK unit tests](https://github.com/IBM/kar/tree/main/examples/python-unit-tests). Extend the set of tests to comprise new Python SDK functionality.

## Basic Service API

### `invoke`

Create a request of the specified `options["method"]` type with an optional body given by `options["body"]` and with content type specified in `options["headers"]` along with other options. The KAR invoke method requires the name of the service and that of the endpoint to be passed in along with the above request options.

Usage for a POST request with Json body:
```
    service_name = "name_of_service_as_string"

    # For a service endpoint named `/endpoint/name` provide:
    endpoint_name = "endpoint/name"

    options = {}
    options["method"] = "POST"
    options["body"] = json.dumps({"name": "John Doe"})
    options["headers"] = {'Content-Type': 'application/json'}

    response = await invoke(service_name, endpoint_name, options)
```

### `tell`

Create an asynchronous call request that does not expect a response back. The method requires the name of the service and that of a service endpoint to be passed along with the request body.

The content type for this type of requests is always `application/json`.

```
    service_name = "name_of_service_as_string"
    data = json.dumps({"greeter": "World"})

    response = await tell(service_name, "test-json-simple", data)
```

### `call`

Create an asynchronous call request. The method requires the name of the service and that of a service endpoint to be passed along with the request body.

The content type for this type of requests is always `application/json`.

```
    service_name = "name_of_service_as_string"
    data = json.dumps({"name": "John", "surname": "Doe"})

    response = await call(service_name, "test-json-object", data)
```

## Basic Actor API

The `KarActor` represents the generic class of a KAR actor. This class is used on the server side to create an actor class:

```
    class MyFirstActor(KarActor):
        def __init__(self):
            pass
```

The `KarActor` class provides the inheriting class with two attributes: `type` and `id` which are used by KAR to uniquely identify an actor. To create a valid actor, it is actually not required to subclass `KarActor`. A valid KAR actor is a class which has the attributes that the `KarActor` class defines, currently these are represented by `type` and `id`. To be future-proof to changes to the `KarActor` class, we recommend using `KarActor` as a base class for your all the actors.

On the client side, the actor class is used to represent a client-side instance of the actor. To create a client-side instance:

```
    client_side_actor_instance = proxy_actor("MyFirstActor", 123)
```

The actor class contains one or more methods that modify/access the state of the actor:

```
class TestActor(KarActor):
    def __init__(self):
        self.name = "John"
        self.surname = "Silver"

    async def set_name(self, name, surname=None):
        self.name = name
        if surname:
            self.surname = surname

    async def get_name(self):
        full_name = [self.name]
        if self.surname:
            full_name.append(self.surname)
        return " ".join(full_name)
```

To call the desired method actor method use the `actor_call` API method ensuring `await` is used. Wrap the call in a an `async` function.

```
    async def call_actor_method():
        actor_instance = proxy_actor("TestActor", 123)
        await actor_call(actor_instance, "set_name", "John", surname="Doe")
        response = await actor_call()
        return response
```

The above function can be called from anywhere in the user code:

```
    asyncio.run(call_actor_method())
```

To remove a proxy actor instance invoke on the respective actor instance:

```
    actor_instance = proxy_actor("TestActor", 123)
    ...
    await actor_remove(actor_instance)
```
