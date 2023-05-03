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

from kar import actor_proxy, actor_call, call
from kar import actor_state_set
from kar import actor_state_submap_set, actor_state_submap_get
from kar import actor_state_submap_get_all, actor_state_get_all
from kar import actor_state_submap_contains, actor_state_submap_remove
from kar import actor_state_contains, actor_state_remove
from kar import actor_state_submap_remove_all, actor_state_submap_remove_some
from kar import actor_state_submap_set_multiple, actor_state_submap_keys
from kar import actor_state_submap_size
import asyncio

service_name = "sdk-test-state-submap"


# -----------------------------------------------------------------------------
# Test state set and get methods
# -----------------------------------------------------------------------------
async def set_and_get_string():
    test_actor = actor_proxy("TestActorSubState", "1")

    # Set actor field `key`:
    await actor_state_submap_set(test_actor, "key", "subkey", "Hello")

    # Read back state of `key` field:
    response = await actor_state_submap_get(test_actor, "key", "subkey")
    assert response == "Hello"


def test_set_and_get_string():
    asyncio.run(set_and_get_string())


# -----------------------------------------------------------------------------
async def set_and_get_field_string():
    test_actor = actor_proxy("TestActorSubState", "2")

    # Set actor field `key`:
    await actor_state_submap_set(test_actor, "key", "subkey",
                                 {"value": "Hello"})

    # Read back state of `key` field:
    response = await actor_state_submap_get(test_actor, "key", "subkey")
    assert response == {"value": "Hello"}


def test_set_and_get_field_string():
    asyncio.run(set_and_get_field_string())


# -----------------------------------------------------------------------------
async def set_and_get_int():
    test_actor = actor_proxy("TestActorSubState", "3")

    # Set actor field `key`:
    await actor_state_submap_set(test_actor, "key", "subkey", 42)

    # Read back state of `key` field:
    response = await actor_state_submap_get(test_actor, "key", "subkey")
    assert response == 42


def test_set_and_get_int():
    asyncio.run(set_and_get_int())


# -----------------------------------------------------------------------------
async def get_all():
    test_actor = actor_proxy("TestActorSubState", "4")

    # Set actor fields:
    await actor_state_submap_set(test_actor, "key", "A", 42)
    await actor_state_submap_set(test_actor, "key", "B", 43)
    await actor_state_submap_set(test_actor, "key", "C", 44)

    # Read back states of all fields:
    response = await actor_state_submap_get_all(test_actor, "key")
    assert response["A"] == 42
    assert response["B"] == 43
    assert response["C"] == 44


def test_get_all():
    asyncio.run(get_all())


# -----------------------------------------------------------------------------
async def get_monexistent_field():
    test_actor = actor_proxy("TestActorSubState", "5")

    # Read back states of all fields:
    response = await actor_state_get_all(test_actor)

    assert response == {}

    # Read back states of all fields:
    response = await actor_state_submap_get_all(test_actor, "key")

    assert response == {}

    # Try to access a specific submap field that doesn't exist:
    response = await actor_state_submap_get(test_actor, "key", "A")

    assert response.status_code == 200
    assert response.content.decode("utf8") == ""


def test_get_monexistent_field():
    asyncio.run(get_monexistent_field())


# -----------------------------------------------------------------------------
async def set_and_get_via_actor():
    test_actor = actor_proxy("TestActorSubState", "6")

    # Set actor field `key`:
    await actor_call(test_actor, "set_persistent_field", "greeting", "Hello")

    # Read back state of `key` field:
    response = await actor_call(test_actor, "get_persistent_field", "greeting")
    assert response == "Hello"


def test_set_and_get_via_actor():
    asyncio.run(set_and_get_via_actor())


# -----------------------------------------------------------------------------
async def contains_field():
    test_actor = actor_proxy("TestActorSubState", "7")

    # Non-existent field.
    assert await actor_state_contains(test_actor, "field") is False

    # Add field:
    await actor_state_set(test_actor, "field", 42)

    # Check field exists:
    assert await actor_state_contains(test_actor, "field") is True

    # Add submap:
    await actor_state_submap_set(test_actor, "field", "key", 43)

    # Check field exists:
    assert await actor_state_submap_contains(test_actor, "field",
                                             "key") is True

    # Fetch value:
    assert await actor_state_submap_get(test_actor, "field", "key") == 43

    # Remove field:
    await actor_state_remove(test_actor, 'field')


def test_contains_field():
    asyncio.run(contains_field())


# -----------------------------------------------------------------------------
async def check_remove_field():
    test_actor = actor_proxy("TestActorSubState", "8")

    # Add field:
    await actor_state_set(test_actor, "field", 42)

    # Check field exists:
    assert await actor_state_contains(test_actor, "field") is True

    # Add sub-field:
    await actor_state_submap_set(test_actor, "field", "key", 42)

    # Check sub-field exists:
    assert await actor_state_submap_contains(test_actor, "field",
                                             "key") is True

    # Remove sub-field:
    await actor_state_submap_remove(test_actor, "field", "key")

    # Check sub-field does not exist:
    assert await actor_state_submap_contains(test_actor, "field",
                                             "key") is False

    # Remove field:
    await actor_state_remove(test_actor, "field")


def test_check_remove_field():
    asyncio.run(check_remove_field())


# -----------------------------------------------------------------------------
async def check_remove_field_twice():
    test_actor = actor_proxy("TestActorSubState", "9")

    # Add field:
    await actor_state_set(test_actor, "field", 42)

    # Check field exists:
    assert await actor_state_contains(test_actor, "field") is True

    # Remove field:
    await actor_state_remove(test_actor, 'field')

    # Check field does not exist:
    assert await actor_state_contains(test_actor, "field") is False

    # Remove field again:
    await actor_state_remove(test_actor, 'field')


def test_check_remove_field_twice():
    asyncio.run(check_remove_field_twice())


# -----------------------------------------------------------------------------
async def remove_all():
    test_actor = actor_proxy("TestActorSubState", "10")

    # Add field:
    await actor_state_set(test_actor, "field1", 42)

    # Add submap keys:
    await actor_state_submap_set(test_actor, "field1", "key1", 42)
    await actor_state_submap_set(test_actor, "field1", "key2", 42)
    await actor_state_submap_set(test_actor, "field1", "key3", 42)

    # Check field exists:
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key1") is True
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key2") is True
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key3") is True

    # Remove field:
    await actor_state_submap_remove_all(test_actor, "field1")

    # Check fields do not exist:
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key1") is False
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key2") is False
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key3") is False

    # Remove field:
    await actor_state_remove(test_actor, "field1")


def test_remove_all():
    asyncio.run(remove_all())


# -----------------------------------------------------------------------------
async def remove_some():
    test_actor = actor_proxy("TestActorSubState", "11")

    # Add field:
    await actor_state_set(test_actor, "field1", 42)

    # Add subkeys:
    await actor_state_submap_set(test_actor, "field1", "key1", 43)
    await actor_state_submap_set(test_actor, "field1", "key2", 44)
    await actor_state_submap_set(test_actor, "field1", "key3", 45)

    # Check subkeys exists:
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key1") is True
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key2") is True
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key3") is True

    # Remove subkeys:
    response = await actor_state_submap_remove_some(test_actor, "field1",
                                                    ["key1", "key3"])

    # Check outcome:
    assert response == 2

    # Check subkey changes:
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key1") is False
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key2") is True
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key3") is False

    # Remove subkey:
    response = await actor_state_submap_remove_some(test_actor, "field1",
                                                    ["key2"])

    # Check outcome:
    assert response == 1

    # Check subkey was removed:
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key2") is False

    # Remove field:
    await actor_state_remove(test_actor, "field1")


def test_remove_some():
    asyncio.run(remove_some())


# -----------------------------------------------------------------------------
async def remove_some_twice():
    test_actor = actor_proxy("TestActorSubState", "12")

    # Add field:
    await actor_state_set(test_actor, "field1", 42)

    # Add subkeys:
    await actor_state_submap_set(test_actor, "field1", "key1", 43)
    await actor_state_submap_set(test_actor, "field1", "key2", 44)
    await actor_state_submap_set(test_actor, "field1", "key3", 45)

    # Check subkeys exists:
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key1") is True
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key2") is True
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key3") is True

    # Remove fields:
    response = await actor_state_submap_remove_some(test_actor, "field1",
                                                    ["key1", "key3"])

    # Check outcome:
    assert response == 2

    # Check subkey changes:
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key1") is False
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key2") is True
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key3") is False

    # Remove fields again:
    response = await actor_state_submap_remove_some(test_actor, "field1",
                                                    ["key1", "key3"])

    # Check outcome:
    assert response == 0

    # Remove field:
    response = await actor_state_submap_remove_some(test_actor, "field1",
                                                    ["key2"])

    # Check outcome:
    assert response == 1

    # Check field was removed:
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key2") is False

    # Remove field again:
    response = await actor_state_submap_remove_some(test_actor, "field1",
                                                    ["key2"])

    # Check outcome:
    assert response == 0


def test_remove_some_twice():
    asyncio.run(remove_some_twice())


# -----------------------------------------------------------------------------
async def add_multiple():
    test_actor = actor_proxy("TestActorSubState", "13")

    # Add field:
    await actor_state_set(test_actor, "field1", 42)

    # Add sub-fields:
    response = await actor_state_submap_set_multiple(test_actor,
                                                     "field1",
                                                     state={
                                                         "key1": 42,
                                                         "key2": 42,
                                                         "key3": 42
                                                     })

    # Check outcome:
    response == 3

    # Add sub-fields:
    response = await actor_state_submap_set_multiple(test_actor,
                                                     "field1",
                                                     state={
                                                         "key4": 42,
                                                         "key5": 42,
                                                     })

    # Check outcome:
    response == 2

    # Check sub-fields:
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key1") is True
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key2") is True
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key3") is True
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key4") is True
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key5") is True

    # Remove all sub-fields:
    response = await actor_state_submap_remove_all(test_actor, "field1")

    # Check sub-fields:
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key1") is False
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key2") is False
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key3") is False
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key4") is False
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key5") is False

    # Remove field:
    await actor_state_remove(test_actor, "field1")


def test_add_multiple():
    asyncio.run(add_multiple())


# -----------------------------------------------------------------------------
async def set_same_field():
    test_actor = actor_proxy("TestActorSubState", "14")

    # Add sub-field:
    await actor_state_submap_set(test_actor, "field1", "key1", 42)

    # Check sub-field exists:
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key1") is True

    # Check sub-field exists:
    assert await actor_state_submap_get(test_actor, "field1", "key1") == 42

    # Add fields:
    await actor_state_submap_set(test_actor, "field1", "key1", 43)

    # Check field exists:
    assert await actor_state_submap_get(test_actor, "field1", "key1") == 43


def test_set_same_field():
    asyncio.run(set_same_field())


# -----------------------------------------------------------------------------
async def list_keys():
    test_actor = actor_proxy("TestActorSubState", "15")

    # Add sub-fields:
    await actor_state_submap_set(test_actor, "field1", "key1", 42)
    await actor_state_submap_set(test_actor, "field1", "key2", 43)
    await actor_state_submap_set(test_actor, "field1", "key3", 44)

    # Check sub-fields exist:
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key1") is True
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key2") is True
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key3") is True

    # Check sub-fields exist:
    assert await actor_state_submap_get(test_actor, "field1", "key1") == 42
    assert await actor_state_submap_get(test_actor, "field1", "key2") == 43
    assert await actor_state_submap_get(test_actor, "field1", "key3") == 44

    # Get all sub-keys:
    response = await actor_state_submap_keys(test_actor, "field1")

    # Check keys:
    assert response == ['key1', 'key2', 'key3']


def test_list_keys():
    asyncio.run(list_keys())


# -----------------------------------------------------------------------------
async def count_keys():
    test_actor = actor_proxy("TestActorSubState", "16")

    # Add sub-fields:
    await actor_state_submap_set(test_actor, "field1", "key1", 42)
    await actor_state_submap_set(test_actor, "field1", "key2", 43)
    await actor_state_submap_set(test_actor, "field1", "key3", 44)

    # Check sub-fields exist:
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key1") is True
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key2") is True
    assert await actor_state_submap_contains(test_actor, "field1",
                                             "key3") is True

    # Check sub-fields exist:
    assert await actor_state_submap_get(test_actor, "field1", "key1") == 42
    assert await actor_state_submap_get(test_actor, "field1", "key2") == 43
    assert await actor_state_submap_get(test_actor, "field1", "key3") == 44

    # Get all sub-keys:
    response = await actor_state_submap_size(test_actor, "field1")

    # Check keys:
    assert response == 3


def test_count_keys():
    asyncio.run(count_keys())


# -----------------------------------------------------------------------------
# Shutdown server gracefully
# -----------------------------------------------------------------------------
async def shutdown():
    response = await call(service_name, "shutdown", None)
    assert response.status_code == 200


def teardown_module():
    asyncio.run(shutdown())
