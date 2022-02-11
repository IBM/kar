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

from kar import actor_runtime, KarActor
import os

# KAR app port
if os.getenv("KAR_APP_PORT") is None:
    raise RuntimeError("KAR_APP_PORT must be set. Aborting.")

kar_app_port = os.getenv("KAR_APP_PORT")

# KAR app host
kar_app_host = '127.0.0.1'
if os.getenv("KAR_APP_HOST") is not None:
    kar_app_host = os.getenv("KAR_APP_HOST")


# Actors are represented by classes that extend the KAR's KarActor
# class.
class FamousActor(KarActor):

    # Kar only supports constructors without arguments. Use methods
    # to update actor state.
    def __init__(self):
        self.name = None
        self.movies = 0

    def set_name(self, name):
        self.name = name

    def get_name(self):
        return self.name

    def add_movie(self):
        self.movies += 1

    def get_movies(self):
        return self.movies


if __name__ == '__main__':
    # Register actor type with the KAR runtime.
    app = actor_runtime([FamousActor])

    # Run the actor server.
    app.run(host=kar_app_host, port=kar_app_port)
