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

from kar import actor_call, actor_proxy, base_call
from kar import events_create_topic, events_delete_topic, events_publish
import httpx
import asyncio
import time
import json


# -----------------------------------------------------------------------------
async def create_delete_topic():
    # Create topic:
    response = await events_create_topic("testtopic1")

    # Check response code:
    assert response.status_code == 201

    # Delete topic:
    response = await events_delete_topic("testtopic1")

    # Check response:
    assert response == "OK"

    time.sleep(2)


def test_actor_create_delete_topic():
    asyncio.run(create_delete_topic())


# -----------------------------------------------------------------------------
async def pub_sub():
    # Topic:
    topic = "testtopic2"

    # Actor instance:
    test_actor = actor_proxy("TestActorEvents", 1)

    # Create topic:
    response = await events_create_topic(topic)

    # Check response code:
    assert response == "OK" or response == "Already existed" or \
        response.status_code == 201

    # Setup subscriber on the server side:
    response = await actor_call(test_actor, "subscribe", topic)
    assert response == 'OK' or response == ''

    # Event data:
    event = json.dumps({"name": "John", "surname": "Doe"})

    # Publish event:
    response = await events_publish(topic, event)
    assert response == "OK"

    # Delete topic:
    response = await events_delete_topic(topic)
    assert response == "OK"

    time.sleep(2)


def test_actor_pub_sub():
    asyncio.run(pub_sub())


# -----------------------------------------------------------------------------
async def get_cancel_subscriptions():
    # Topic:
    topic = "testtopic3"

    # Actor instance:
    test_actor = actor_proxy("TestActorEvents", 2)

    # Create topic:
    response = await events_create_topic(topic)

    # Check response code:
    assert response == "OK" or response == "Already existed" or \
        response.status_code == 201

    # Setup subscriber on the server side:
    response = await actor_call(test_actor, "subscribe", topic)
    assert response == 'OK' or response == ''

    # Get subscription:
    response = await actor_call(test_actor, "get_subscription")
    assert len(response) == 1

    subscription = response[0]
    assert subscription["id"] == topic
    assert subscription["actor"]["Type"] == "TestActorEvents"
    assert subscription["actor"]["ID"] == "2"

    # Event data:
    event = json.dumps({"name": "John", "surname": "Doe"})

    # Publish event:
    response = await events_publish(topic, event)
    assert response == "OK"

    # Cancel subscription:
    response = await actor_call(test_actor, "cancel_subscription")
    assert response == '1'

    # Get subscription:
    response = await actor_call(test_actor, "get_subscription")
    assert response == []

    # Delete topic:
    response = await events_delete_topic(topic)
    assert response == "OK"

    time.sleep(2)


def test_actor_get_cancel_subscriptions():
    asyncio.run(get_cancel_subscriptions())


# -----------------------------------------------------------------------------
async def shutdown_server():
    # Shutdown server:
    try:
        response = await base_call("sdk-test-events", "shutdown", None)
    except httpx.HTTPStatusError:
        assert False

    assert response.status_code == 200
    assert response.content.decode("utf8") == "shutting down"


def test_actor_shutdown_server():
    asyncio.run(shutdown_server())
