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

from kar import call, tell, invoke, async_call
import asyncio
import json

service_name = "sdk-test-services"


# -----------------------------------------------------------------------------
# Text request and responses for CALL
# -----------------------------------------------------------------------------
async def text_simple(request):
    response = await request(service_name, "test-text-simple", "World")
    assert response == "Hello World"


def test_text_simple():
    asyncio.run(text_simple(call))


# -----------------------------------------------------------------------------
async def text_structured(request):
    response = await request(service_name, "test-text-structured", "World")
    assert response == "Hello World"


def test_structured():
    asyncio.run(text_structured(call))


# -----------------------------------------------------------------------------
async def text_structured_auto(request):
    response = await request(service_name, "test-text-structured-auto",
                             "World")
    assert response == "Hello World"


def test_text_structured_auto():
    asyncio.run(text_structured_auto(call))


# -----------------------------------------------------------------------------
async def text_structured_generic(request):
    response = await request(service_name, "test-text-structured-generic",
                             "World")
    assert response.status_code == 200
    assert response.content.decode("utf8") == "Hello World"


def test_text_structured_generic():
    asyncio.run(text_structured_generic(call))


# -----------------------------------------------------------------------------
async def text_simple_async():
    response = await async_call(service_name, "test-text-simple", "World")
    assert response == "Hello World"


def test_text_simple_async():
    asyncio.run(text_simple_async())


# -----------------------------------------------------------------------------
async def text_structured_generic_auto(request):
    response = await request(service_name, "test-text-structured-generic-auto",
                             "World")
    assert response.status_code == 200
    assert response.content.decode("utf8") == "Hello World"


def test_text_structured_generic_auto():
    asyncio.run(text_structured_generic_auto(call))


# -----------------------------------------------------------------------------
# JSON request and responses
# -----------------------------------------------------------------------------
async def text_to_json(request):
    response = await request(service_name, "test-text-to-json", "World")
    assert response["message"] == "Hello World"


def test_text_to_json():
    asyncio.run(text_to_json(call))


# -----------------------------------------------------------------------------


async def json_simple(request):
    data = json.dumps({"greeter": "World"})
    response = await request(service_name, "test-json-simple", data)
    assert response["message"] == "Hello World"


def test_json_simple():
    asyncio.run(json_simple(call))


# -----------------------------------------------------------------------------
async def json_structured(request):
    data = json.dumps({"greeter": "World"})
    response = await request(service_name, "test-json-structured", data)
    assert response["message"] == "Hello World"


def test_json_structured():
    asyncio.run(json_structured(call))


# -----------------------------------------------------------------------------
async def json_structured_auto(request):
    data = json.dumps({"greeter": "World"})
    response = await request(service_name, "test-json-structured-auto", data)
    assert response["message"] == "Hello World"


def test_json_structured_auto():
    asyncio.run(json_structured_auto(call))


# -----------------------------------------------------------------------------
async def json_object(request):
    data = json.dumps({"name": "John", "surname": "Doe"})
    response = await request(service_name, "test-json-object", data)
    assert response["message"] == "Hello John Doe"


def test_json_object():
    asyncio.run(json_object(call))


# -----------------------------------------------------------------------------
async def json_generic(request):
    data = json.dumps({"greeter": "World"})
    response = await request(service_name, "test-json-generic", data)
    assert response.status_code == 200
    content = response.json()
    assert content["message"] == "Hello World"


def test_json_generic():
    asyncio.run(json_generic(call))


# -----------------------------------------------------------------------------
async def text_to_json_async():
    response = await async_call(service_name, "test-text-to-json", "World")
    assert response["message"] == "Hello World"


def test_text_to_json_async():
    asyncio.run(text_to_json_async())


# -----------------------------------------------------------------------------
async def json_generic_auto(request):
    data = json.dumps({"greeter": "World"})
    response = await request(service_name, "test-json-generic-auto", data)
    content = json.loads(response)
    assert content["message"] == "Hello World"


def test_json_generic_auto():
    asyncio.run(json_generic_auto(call))


# -----------------------------------------------------------------------------
# Text request and responses for TELL
# -----------------------------------------------------------------------------
async def text_simple_tell(request):
    response = await request(service_name, "test-text-simple", "World")
    assert response == "OK"


def test_text_simple_tell():
    asyncio.run(text_simple_tell(tell))


# -----------------------------------------------------------------------------
async def json_simple_tell(request):
    data = json.dumps({"greeter": "World"})
    response = await request(service_name, "test-json-simple", data)
    assert response == "OK"


def test_json_simple_tell():
    asyncio.run(json_simple_tell(tell))


# -----------------------------------------------------------------------------
# Text request and responses for INVOKE
# -----------------------------------------------------------------------------
async def text_simple_invoke(request):
    options = {}
    options["body"] = "World"
    options["method"] = "POST"
    response = await request(service_name, "test-text-simple", options)
    assert response == "Hello World"


def test_text_simple_invoke():
    asyncio.run(text_simple_invoke(invoke))


# -----------------------------------------------------------------------------
async def json_simple_invoke_post(request):
    options = {}
    options["body"] = json.dumps({"greeter": "World"})
    options["method"] = "POST"
    options["headers"] = {'Content-Type': 'application/json'}
    response = await request(service_name, "test-json-simple", options)
    assert response["message"] == "Hello World"


def test_json_simple_invoke_post():
    asyncio.run(json_simple_invoke_post(invoke))


# -----------------------------------------------------------------------------
async def json_simple_invoke_get(request):
    options = {}
    options["method"] = "GET"
    options["headers"] = {'Content-Type': 'application/json'}
    response = await request(service_name, "test-get", options)
    assert response == "OK"


def test_json_simple_invoke_get():
    asyncio.run(json_simple_invoke_get(invoke))


# -----------------------------------------------------------------------------
async def json_simple_invoke_delete(request):
    options = {}
    options["method"] = "DELETE"
    options["headers"] = {'Content-Type': 'application/json'}
    response = await request(service_name, "test-delete", options)
    assert response == "OK"


def test_json_simple_invoke_delete():
    asyncio.run(json_simple_invoke_delete(invoke))


# -----------------------------------------------------------------------------
async def json_simple_invoke_put(request):
    options = {}
    options["body"] = json.dumps({"greeter": "World"})
    options["method"] = "PUT"
    options["headers"] = {'Content-Type': 'application/json'}
    response = await request(service_name, "test-put", options)
    assert response == "Hello World"


def test_json_simple_invoke_put():
    asyncio.run(json_simple_invoke_put(invoke))


# -----------------------------------------------------------------------------
async def json_simple_invoke_head(request):
    options = {}
    options["method"] = "HEAD"
    response = await request(service_name, "test-head", options)
    assert response.status_code == 200


def test_json_simple_invoke_head():
    asyncio.run(json_simple_invoke_head(invoke))


# -----------------------------------------------------------------------------
# Shutdown server gracefully
# -----------------------------------------------------------------------------
async def shutdown(request):
    response = await request(service_name, "shutdown", None)
    assert response.status_code == 200


def teardown_module():
    asyncio.run(shutdown(call))
