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

from hypercorn.config import Config
from hypercorn.asyncio import serve
from fastapi import FastAPI, Request
from fastapi.responses import Response, JSONResponse, PlainTextResponse
from pydantic import BaseModel
import os
import asyncio
import json

# KAR app port
if os.getenv("KAR_APP_PORT") is None:
    raise RuntimeError("KAR_APP_PORT must be set. Aborting.")

kar_app_port = os.getenv("KAR_APP_PORT")

# KAR app host
kar_app_host = '127.0.0.1'
if os.getenv("KAR_APP_HOST") is not None:
    kar_app_host = os.getenv("KAR_APP_HOST")

# Setup server:
config = Config()
config.bind = [f"{kar_app_host}:{kar_app_port}"]
config.alpn_protocols = ['h2']

# Shutdown event:
shutdown_event = asyncio.Event()

# Create app:
app = FastAPI()


# Test class:
class TestPerson(BaseModel):
    name: str
    surname: str


# -----------------------------------------------------------------------------
# Text request and responses
# -----------------------------------------------------------------------------
@app.post('/test-text-simple')
async def test_call_text_simple(request: Request):
    body = await request.body()
    body = body.decode("utf8")
    return " ".join(["Hello", body])


@app.post('/test-text-structured', response_class=PlainTextResponse)
async def test_call_text_structured(request: Request):
    body = await request.body()
    body = body.decode("utf8")
    response_message = " ".join(["Hello", body])
    return PlainTextResponse(status_code=200, content=response_message)


@app.post('/test-text-structured-auto', response_class=PlainTextResponse)
async def test_call_text_structured_auto(request: Request):
    body = await request.body()
    body = body.decode("utf8")
    response_message = " ".join(["Hello", body])
    return response_message


@app.post('/test-text-structured-generic', response_class=Response)
async def test_call_text_structured_generic(request: Request):
    body = await request.body()
    body = body.decode("utf8")
    response_message = " ".join(["Hello", body])
    return Response(status_code=200, content=response_message)


@app.post('/test-text-structured-generic-auto', response_class=Response)
async def test_call_text_structured_generic_auto(request: Request):
    body = await request.body()
    body = body.decode("utf8")
    response_message = " ".join(["Hello", body])
    return response_message


# -----------------------------------------------------------------------------
# JSON requests and responses
# -----------------------------------------------------------------------------
@app.post('/test-text-to-json')
async def test_call_text_to_json(request: Request):
    body = await request.body()
    body = body.decode("utf8")
    response_message = " ".join(["Hello", body])
    return {"message": response_message}


@app.post('/test-json-simple')
async def test_call_json_simple(request: Request):
    body = await request.json()
    response_message = " ".join(["Hello", body["greeter"]])
    return {"message": response_message}


@app.post('/test-json-structured')
async def test_call_json_structured(request: Request,
                                    response_class=JSONResponse):
    body = await request.json()
    response_message = " ".join(["Hello", body["greeter"]])
    return JSONResponse(status_code=200, content={"message": response_message})


@app.post('/test-json-structured-auto')
async def test_call_json_structured_auto(request: Request,
                                         response_class=JSONResponse):
    body = await request.json()
    response_message = " ".join(["Hello", body["greeter"]])
    return {"message": response_message}


@app.post('/test-json-object', response_class=JSONResponse)
async def test_call_json_object(person: TestPerson):
    response = " ".join(["Hello", person.name, person.surname])
    response_message = {"message": response}
    return JSONResponse(status_code=200, content=response_message)


@app.post('/test-json-generic')
async def test_call_json_generic(request: Request, response_class=Response):
    body = await request.json()
    response_message = " ".join(["Hello", body["greeter"]])
    return Response(status_code=200,
                    content=json.dumps({"message": response_message}))


@app.post('/test-json-generic-auto')
async def test_call_json_generic_auto(request: Request,
                                      response_class=Response):
    body = await request.json()
    response_message = " ".join(["Hello", body["greeter"]])
    return json.dumps({"message": response_message})


# -----------------------------------------------------------------------------
# HTTP requests
# -----------------------------------------------------------------------------
@app.get('/test-get', response_class=JSONResponse)
async def test_text_get(request: Request):
    return JSONResponse(status_code=200, content="OK")


@app.head('/test-head')
async def test_text_head(request: Request):
    return Response(status_code=200)


@app.delete('/test-delete', response_class=JSONResponse)
async def test_text_delete(request: Request):
    return JSONResponse(status_code=200, content="OK")


@app.put('/test-put', response_class=JSONResponse)
async def test_text_put(request: Request):
    body = await request.json()
    response_message = " ".join(["Hello", body["greeter"]])
    return JSONResponse(status_code=200, content=response_message)


# -----------------------------------------------------------------------------
# HTTP version tests
# -----------------------------------------------------------------------------
@app.post('/test-http-version', response_class=JSONResponse)
async def test_call_json(request: Request):
    return JSONResponse(status_code=200,
                        content={"version": request["http_version"]})


# -----------------------------------------------------------------------------
# Server shutdown method
# -----------------------------------------------------------------------------
@app.post('/shutdown')
async def shutdown():
    shutdown_event.set()
    return Response(status_code=200, content="shutting down")


if __name__ == '__main__':
    # Run the actor server.
    loop = asyncio.get_event_loop()
    loop.run_until_complete(
        serve(app, config, shutdown_trigger=shutdown_event.wait))
    loop.close()
