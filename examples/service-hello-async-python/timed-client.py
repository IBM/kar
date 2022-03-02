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

import os
import json
import httpx
import asyncio
import time

# KAR runtime port
if os.getenv("KAR_RUNTIME_PORT") is None:
    raise RuntimeError("KAR_RUNTIME_PORT must be set. Aborting.")

kar_runtime_port = os.getenv("KAR_RUNTIME_PORT")


def kar_endpoint(request_type, service, route):
    return f"http://localhost:{kar_runtime_port}/kar/v1/service/" \
        f"{service}/{request_type}/{route}"


async def client():
    async with httpx.AsyncClient(http1=False, http2=True,
                                 timeout=15.0) as client:
        # Setup data:
        data = json.dumps({"name": "John", "surname": "Doe"})
        headers = {'Content-Type': 'application/json'}

        # Timing run (Sequential performance):
        start = time.time()
        response1 = client.post(kar_endpoint("call", "async-server",
                                             "helloJson"),
                                data=data,
                                headers=headers)
        response2 = client.post(kar_endpoint("call", "async-server",
                                             "helloJson"),
                                data=data,
                                headers=headers)
        response1 = await response1
        response2 = await response2
        print("Time =", time.time() - start)
        json_content = json.loads(response1.content.decode("utf8"))
        print(json_content["message"])
        json_content = json.loads(response2.content.decode("utf8"))
        print(json_content["message"])

        # Timing run with GATHER (parallel performance):
        start = time.time()
        response1 = client.post(kar_endpoint("call", "async-server",
                                             "helloJson"),
                                data=data,
                                headers=headers)
        response2 = client.post(kar_endpoint("call", "async-server",
                                             "helloJson"),
                                data=data,
                                headers=headers)
        responses = await asyncio.gather(response1, response2)
        print("Time =", time.time() - start)
        json_content = json.loads(responses[0].content.decode("utf8"))
        print(json_content["message"])
        json_content = json.loads(responses[1].content.decode("utf8"))
        print(json_content["message"])

        # Timing run with TASKS (parallel performance):
        start = time.time()
        response1 = asyncio.create_task(
            client.post(kar_endpoint("call", "async-server", "helloJson"),
                        data=data,
                        headers=headers))
        response2 = asyncio.create_task(
            client.post(kar_endpoint("call", "async-server", "helloJson"),
                        data=data,
                        headers=headers))
        response1 = await response1
        response2 = await response2
        print("Time =", time.time() - start)
        json_content = json.loads(response1.content.decode("utf8"))
        print(json_content["message"])
        json_content = json.loads(response2.content.decode("utf8"))
        print(json_content["message"])


def main():
    asyncio.run(client())


if __name__ == "__main__":
    main()
