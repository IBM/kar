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

from kar import actor_call, actor_proxy, tell, shutdown
import asyncio


async def test_actor_call():
    famous_actor = actor_proxy("FamousActor", "56868876")

    # Set a field value:
    await actor_call(famous_actor,
                     "set_name",
                     "John",
                     suffix="Jr.",
                     surname="Smith")

    # Get actor name value:
    response = await actor_call(famous_actor, "get_name")
    print(response)

    # Add movies:
    for i in range(120):
        await actor_call(famous_actor, "add_movie")

    # Get number of movies:
    response = await actor_call(famous_actor, "get_movies")
    print("Movies:", response)

    # await actor_call(famous_actor, "exit")
    await tell("actor-server-service", "shutdown", None)

    print("SUCCESS")

    # Shutdown the sidecar:
    await shutdown()


def main():
    asyncio.run(test_actor_call())
    return 0


if __name__ == "__main__":
    main()
