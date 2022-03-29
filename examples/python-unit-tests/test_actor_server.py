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

from kar import actor_call, actor_proxy, base_call, actor_remove
from kar import actor_async_call, actor_root_call
from kar import actor_schedule_reminder, actor_get_reminder
from kar import actor_cancel_reminder
import httpx
import asyncio
import traceback
import datetime


# -----------------------------------------------------------------------------
async def set_actor_field():
    famous_actor = actor_proxy("TestActor", "1")

    # Set actor state via actor method:
    await actor_call(famous_actor,
                     "set_name",
                     "John",
                     suffix="Jr.",
                     surname="Smith")

    # Retrieve field value:
    response = await actor_call(famous_actor, "get_name")
    assert response == "John Smith Jr."


def test_actor_set_and_get():
    asyncio.run(set_actor_field())


# -----------------------------------------------------------------------------
async def set_non_existent_actor_field():
    famous_actor = actor_proxy("TestActor", "2")

    # Call non-existent actor method:
    try:
        await actor_call(famous_actor,
                         "set_full_name",
                         "John",
                         suffix="Jr.",
                         surname="Smith")
    except httpx.HTTPStatusError as error:
        assert error.response.status_code == 404
        assert error.response.content.decode(
            "utf8") == "no set_full_name in actor with type TestActor and id 2"
        return

    assert False


def test_actor_non_existent_method():
    asyncio.run(set_non_existent_actor_field())


# -----------------------------------------------------------------------------
async def wrong_arguments():
    famous_actor = actor_proxy("TestActor", "3")

    # Call actor method with wrong arguments:
    try:
        response = await actor_call(famous_actor, "set_name")
    except httpx.HTTPStatusError:
        assert False
    error_msg = "set_name() missing 1 required positional argument: 'name'"
    assert error_msg in response


def test_actor_wrong_arguments():
    asyncio.run(wrong_arguments())


# -----------------------------------------------------------------------------
async def missing_actor():
    famous_actor = actor_proxy("NonexistentActor", "1")

    # Call actor method with wrong arguments:
    try:
        await actor_call(famous_actor, "set_name")
    except httpx.ReadTimeout:
        error_msg = "ReadTimeout"
        assert error_msg in traceback.format_exc()
        return
    assert False


def test_actor_missing_actor():
    # Temporarily disabled test since there is no established way to determine
    # if an actor exists or not.
    # asyncio.run(missing_actor())
    pass


# -----------------------------------------------------------------------------
async def non_json_content():
    famous_actor = actor_proxy("TestActor", "4")

    # Call existent actor method with non-json content-type:
    try:
        await actor_call(famous_actor, "not_callable")
    except httpx.HTTPStatusError as error:
        assert error.response.status_code == 404
        assert error.response.content.decode(
            "utf8") == "no not_callable in actor with type TestActor and id 4"
        return

    assert False


def test_actor_non_json_content():
    asyncio.run(non_json_content())


# -----------------------------------------------------------------------------
async def not_callable():
    famous_actor = actor_proxy("TestActor", "5")

    # Call existent actor non-callable attribute:
    try:
        await actor_call(famous_actor, "not_callable")
    except httpx.HTTPStatusError as error:
        assert error.response.status_code == 404
        assert error.response.content.decode(
            "utf8") == "no not_callable in actor with type TestActor and id 5"
        return

    assert False


def test_actor_not_callable():
    asyncio.run(not_callable())


# -----------------------------------------------------------------------------
async def head_actor_call():
    # The actor name and type was registered with the server at launch time
    # using the `-actors` flag on the `kar run` command line:
    #
    #  kar run [...] -actors TestActor [...]
    #
    actor_name = "TestActor"

    # Call existent actor head method. The actual HEAD method can only be
    # called by the server itself. In our test server we expose a method which
    # performs a HEAD request to check if actor was registered (which it was).
    try:
        response = await base_call("sdk-test", f"check/{actor_name}", None)
    except httpx.HTTPStatusError:
        assert False

    assert response.status_code == 200


def test_actor_head_actor_call():
    asyncio.run(head_actor_call())


# -----------------------------------------------------------------------------
async def remove_actor():
    famous_actor = actor_proxy("TestActor", "6")

    # Remove actor:
    try:
        response = await actor_remove(famous_actor)
    except httpx.HTTPStatusError:
        assert False

    assert response == "OK"


def test_actor_remove_actor():
    asyncio.run(remove_actor())


# -----------------------------------------------------------------------------
async def head_nonexistent_actor_call():
    # The actor name and type was not registered with the server at launch
    # time.
    actor_name = "NonexistentActor"

    # Call existent actor head method. The actual HEAD method can only be
    # called by the server itself. In our test server we expose a method which
    # performs a HEAD request to check if actor was registered (which it was).
    try:
        response = await base_call("sdk-test", f"check/{actor_name}", None)
    except httpx.HTTPStatusError:
        assert False

    assert response.status_code == 404


def test_actor_head_nonexistent_actor_call():
    asyncio.run(head_nonexistent_actor_call())


# -----------------------------------------------------------------------------
async def health_check():
    # Test if actor server is up and running:
    try:
        response = await base_call("sdk-test", "healthy", None)
    except httpx.HTTPStatusError:
        assert False

    assert response.status_code == 200
    assert response.content.decode("utf8") == "Peachy Keen!"


def test_actor_health_check():
    asyncio.run(health_check())


# -----------------------------------------------------------------------------
async def set_actor_field_async():
    famous_actor = actor_proxy("TestActor", "7")

    # Set actor state via actor method:
    await actor_async_call(famous_actor,
                           "set_name",
                           "John",
                           suffix="Jr.",
                           surname="Smith")

    # Retrieve field value:
    response = await actor_async_call(famous_actor, "get_name")
    assert response == "John Smith Jr."


def test_set_actor_field_async():
    asyncio.run(set_actor_field_async())


# -----------------------------------------------------------------------------
async def actor_root_call_access():
    famous_actor = actor_proxy("TestActor", "8")

    # Set actor state via actor method:
    await actor_root_call(famous_actor,
                          "set_name",
                          "John",
                          suffix="Jr.",
                          surname="Smith")

    # Retrieve field value:
    response = await actor_root_call(famous_actor, "get_name")
    assert response == "John Smith Jr."


def test_actor_root_call_access():
    asyncio.run(actor_root_call_access())


# -----------------------------------------------------------------------------
async def actor_reminders_schedule():
    famous_actor = actor_proxy("TestActor", "9")

    # Current time:
    current = datetime.datetime.now()

    # Future time +5s:
    future_time = current + datetime.timedelta(0, 5)
    formatted_time = future_time.astimezone().isoformat()
    print(formatted_time)

    # Schedule reminder:
    await actor_schedule_reminder(famous_actor,
                                  "set_name", {
                                      "id": "default_id_1",
                                      "target_time": formatted_time,
                                      "period": "5s"
                                  },
                                  "John",
                                  suffix="Jr.",
                                  surname="Smith")

    # Check that the reminder was registered:
    response = await actor_get_reminder(famous_actor)

    # A list of reminders is returned, in this case there is only 1:
    assert len(response) == 1
    reminder = response[0]
    assert reminder["Actor"] == {'ID': '9', 'Type': 'TestActor'}
    assert reminder["id"] == "default_id_1"
    assert reminder["path"] == "/set_name"

    # Get the reminder by name:
    response = await actor_get_reminder(famous_actor, "default_id_1")
    assert response["Actor"] == {'ID': '9', 'Type': 'TestActor'}
    assert response["id"] == "default_id_1"
    assert response["path"] == "/set_name"

    # Cancel the reminder:
    response = await actor_cancel_reminder(famous_actor, "default_id_1")
    assert response == "1"

    # Get the reminder:
    response = await actor_get_reminder(famous_actor, "default_id_1")
    assert response is None


def test_actor_reminders_schedule():
    asyncio.run(actor_reminders_schedule())


# -----------------------------------------------------------------------------
async def actor_tail_calls():
    famous_actor = actor_proxy("TestActor", "10")

    # Set actor state via actor method:
    await actor_call(famous_actor,
                     "set_name_with_tail_1",
                     "John",
                     suffix="Jr.",
                     surname="Smith")

    response = await actor_call(famous_actor, "get_name")
    assert response == "John Smith Jr."

    # Set actor state via actor method:
    await actor_call(famous_actor,
                     "set_name_with_tail_2",
                     "Bill",
                     suffix="Sr.",
                     surname="Smith")

    response = await actor_call(famous_actor, "get_name")
    assert response == "Bill Smith Sr."

    # Set actor state via actor method:
    await actor_call(famous_actor,
                     "set_name_with_tail_3",
                     "John",
                     suffix="Jr.",
                     surname="Doe")

    response = await actor_call(famous_actor, "get_name")
    assert response == "John Doe Jr."

    # Set actor state via actor method:
    await actor_call(famous_actor,
                     "set_name_with_tail_4",
                     "Bill",
                     suffix="Sr.",
                     surname="Doe")

    response = await actor_call(famous_actor, "get_name")
    assert response == "Bill Doe Sr."


def test_actor_tail_calls():
    asyncio.run(actor_tail_calls())


# -----------------------------------------------------------------------------
async def actor_tail_pet_calls():
    famous_actor = actor_proxy("TestActor", "11")

    pet_actor = actor_proxy("AnotherTestActor", "1")

    # Set pet actor in actor:
    await actor_call(famous_actor, "set_pet", pet_actor.type, pet_actor.id)

    # Set details of pet:
    await actor_call(famous_actor, "set_pet_details", "cat", "Tiddles")

    # Check pet name and type:
    response = await actor_call(pet_actor, "get_pet")
    assert response == "cat: Tiddles"


def test_actor_tail_pet_calls():
    asyncio.run(actor_tail_pet_calls())


# -----------------------------------------------------------------------------
async def actor_tail_pet_individual_calls():
    famous_actor = actor_proxy("TestActor", "12")

    pet_actor = actor_proxy("AnotherTestActor", "2")

    # Set pet actor in actor:
    await actor_call(famous_actor, "set_pet", pet_actor.type, pet_actor.id)

    # Set details of pet:
    await actor_call(famous_actor, "set_pet_name", "Tiddles")
    await actor_call(famous_actor, "set_pet_type", "cat")

    # Check pet name and type:
    response = await actor_call(pet_actor, "get_pet")
    assert response == "cat: Tiddles"


def test_actor_tail_pet_individual_calls():
    asyncio.run(actor_tail_pet_individual_calls())


# -----------------------------------------------------------------------------
async def shutdown_server():
    # Shutdown server:
    try:
        response = await base_call("sdk-test", "shutdown", None)
    except httpx.HTTPStatusError:
        assert False

    assert response.status_code == 200
    assert response.content.decode("utf8") == "shutting down"


def test_actor_shutdown_server():
    asyncio.run(shutdown_server())
