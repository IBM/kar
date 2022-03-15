# Copyright IBM Corporation 2020,2022
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

from kar import actor_proxy, actor_call, call
from kar import actor_state_set, actor_state_get, actor_state_get_all
from kar import actor_state_contains, actor_state_remove
import asyncio

service_name = "sdk-test-state"


# -----------------------------------------------------------------------------
# Test state set and get methods
# -----------------------------------------------------------------------------
async def set_and_get_string():
    test_actor = actor_proxy("TestActorState", "1")

    # Set actor field `key`:
    await actor_state_set(test_actor, "key", "Hello")

    # Read back state of `key` field:
    response = await actor_state_get(test_actor, "key")
    assert response == "Hello"


def test_set_and_get_string():
    asyncio.run(set_and_get_string())


# -----------------------------------------------------------------------------
async def set_and_get_field_string():
    test_actor = actor_proxy("TestActorState", "2")

    # Set actor field `key`:
    await actor_state_set(test_actor, "key", {"value": "Hello"})

    # Read back state of `key` field:
    response = await actor_state_get(test_actor, "key")
    assert response == {"value": "Hello"}


def test_set_and_get_field_string():
    asyncio.run(set_and_get_field_string())


# -----------------------------------------------------------------------------
async def set_and_get_int():
    test_actor = actor_proxy("TestActorState", "3")

    # Set actor field `key`:
    await actor_state_set(test_actor, "key", 42)

    # Read back state of `key` field:
    response = await actor_state_get(test_actor, "key")
    assert response == 42


def test_set_and_get_int():
    asyncio.run(set_and_get_int())


# -----------------------------------------------------------------------------
async def get_all():
    test_actor = actor_proxy("TestActorState", "4")

    # Set actor fields:
    await actor_state_set(test_actor, "A", 42)
    await actor_state_set(test_actor, "B", 43)
    await actor_state_set(test_actor, "C", 44)

    # Read back states of all fields:
    response = await actor_state_get_all(test_actor)
    assert response["A"] == 42
    assert response["B"] == 43
    assert response["C"] == 44


def test_get_all():
    asyncio.run(get_all())


# -----------------------------------------------------------------------------
async def get_monexistent_field():
    test_actor = actor_proxy("TestActorState", "5")

    # Read back states of all fields:
    response = await actor_state_get_all(test_actor)

    assert response == {}

    # Try to access a specific field that doesn't exist:
    response = await actor_state_get(test_actor, "A")

    assert response.status_code == 200
    assert response.content.decode("utf8") == ""


def test_get_monexistent_field():
    asyncio.run(get_monexistent_field())


# -----------------------------------------------------------------------------
async def set_and_get_via_actor():
    test_actor = actor_proxy("TestActorState", "6")

    # Set actor field `key`:
    await actor_call(test_actor, "set_persistent_field", "Hello")

    # Read back state of `key` field:
    response = await actor_call(test_actor, "get_persistent_field")
    assert response == "Hello"


def test_set_and_get_via_actor():
    asyncio.run(set_and_get_via_actor())


# -----------------------------------------------------------------------------
async def contains_field():
    test_actor = actor_proxy("TestActorState", "7")

    # Non-existent field.
    assert await actor_state_contains(test_actor, "field") is False

    # Add field:
    await actor_state_set(test_actor, "field", 42)

    # Check field exists:
    assert await actor_state_contains(test_actor, "field") is True

    # Remove field:
    await actor_state_remove(test_actor, 'field')


def test_contains_field():
    asyncio.run(contains_field())


# -----------------------------------------------------------------------------
async def check_remove_field():
    test_actor = actor_proxy("TestActorState", "8")

    # Add field:
    await actor_state_set(test_actor, "field", 42)

    # Check field exists:
    assert await actor_state_contains(test_actor, "field") is True

    # Remove field:
    await actor_state_remove(test_actor, 'field')

    # Check field does not exist:
    assert await actor_state_contains(test_actor, "field") is False


def test_check_remove_field():
    asyncio.run(check_remove_field())


# -----------------------------------------------------------------------------
# Shutdown server gracefully
# -----------------------------------------------------------------------------
async def shutdown():
    response = await call(service_name, "shutdown", None)
    assert response.status_code == 200


def test_shutdown():
    asyncio.run(shutdown())
