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

from kar import actor_runtime, KarActor
from kar import actor_state_submap_set, actor_state_submap_get
from hypercorn.config import Config
from hypercorn.asyncio import serve
from fastapi import FastAPI, Response
import os
import asyncio

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


# Actors are represented by classes that extend the KAR's KarActor
# class.
class TestActorSubState(KarActor):

    # Kar only supports constructors without arguments. Use methods
    # to update actor state.
    def __init__(self):
        self.name = None
        self.surname = None
        self.suffix = None
        self.movies = 0
        self.not_callable = None

    def set_name(self, name, surname=None, suffix=None):
        self.name = name
        if surname:
            self.surname = surname
        if suffix:
            self.suffix = suffix

    def get_name(self):
        full_name = [self.name]
        if self.surname:
            full_name.append(self.surname)
        if self.suffix:
            full_name.append(self.suffix)
        return " ".join(full_name)

    def add_movie(self):
        self.movies += 1

    async def set_persistent_field(self, subkey, value):
        await actor_state_submap_set(self, "field", subkey, value)

    async def get_persistent_field(self, subkey):
        return await actor_state_submap_get(self, "field", subkey)

    def get_movies(self):
        return self.movies


if __name__ == '__main__':
    # Register actor type with the KAR runtime.
    app = actor_runtime([TestActorSubState], actor_server=app)

    @app.post('/shutdown')
    async def shutdown():
        shutdown_event.set()
        return Response(status_code=200, content="shutting down")

    # Run the actor server.
    loop = asyncio.get_event_loop()
    loop.run_until_complete(
        serve(app, config, shutdown_trigger=shutdown_event.wait))
    loop.close()
