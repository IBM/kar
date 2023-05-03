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

import asyncio
from fastapi import FastAPI, Request
from fastapi.responses import Response, JSONResponse
from pydantic import BaseModel
from hypercorn.config import Config
from hypercorn.asyncio import serve
import sys
import os

# KAR app port
if os.getenv("KAR_APP_PORT") is None:
    raise RuntimeError("KAR_APP_PORT must be set. Aborting.")

kar_app_port = os.getenv("KAR_APP_PORT")

# KAR app host
kar_app_host = '127.0.0.1'
if os.getenv("KAR_APP_HOST") is not None:
    kar_app_host = os.getenv("KAR_APP_HOST")


class Person(BaseModel):
    name: str
    surname: str


app = FastAPI()


@app.post("/helloJson", response_class=JSONResponse)
async def post_hello(person: Person, request: Request):
    greetings_message = " ".join(["Hello", person.name, person.surname])
    print("HTTP version:", request["http_version"], flush=True)
    await asyncio.sleep(5)
    return {"message": greetings_message}


@app.post("/helloText", response_class=Response)
async def post_hello_text(request: Request):
    body = await request.body()
    body = body.decode("utf8")
    greetings_message = " ".join(["Hello", body])
    print("HTTP version:", request["http_version"], flush=True)
    await asyncio.sleep(5)
    return greetings_message


@app.get('/health')
def health_check():
    health_message = "I am healthy"
    print(health_message, file=sys.stderr, flush=True)
    return {"message": health_message}


config = Config()
config.bind = [f"{kar_app_host}:{kar_app_port}"]
config.alpn_protocols = ['h2']

asyncio.run(serve(app, config))
