# Copyright IBM Corporation 2020,2021,2022
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

import requests
import os
import json

# KAR runtime port
if os.getenv("KAR_RUNTIME_PORT") is None:
    raise RuntimeError("KAR_RUNTIME_PORT must be set. Aborting.")

kar_runtime_port = os.getenv("KAR_RUNTIME_PORT")


def kar_endpoint(request_type, service, route):
    return f"http://localhost:{kar_runtime_port}/kar/v1/service/" \
        f"{service}/{request_type}/{route}"


def main():
    # Send plain text message
    data = "John Doe"
    headers = {'Content-Type': 'text/plain'}
    response = requests.post(kar_endpoint("call", "python-greeter",
                                          "helloText"),
                             data=data,
                             headers=headers)
    print(response.content.decode("utf8"))

    # Send JSON message
    data = json.dumps({"name": "John Doe"})
    headers = {'Content-Type': 'application/json'}
    response = requests.post(kar_endpoint("call", "python-greeter",
                                          "helloJson"),
                             data=data,
                             headers=headers)
    json_content = json.loads(response.content.decode("utf8"))
    print(json_content["greetings"])

    # Send health check via GET request
    response = requests.get(kar_endpoint("call", "python-greeter", "health"))
    print(response.content.decode("utf8"))


if __name__ == "__main__":
    main()
