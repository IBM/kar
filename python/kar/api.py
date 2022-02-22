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
#

import aiohttp
import asyncio
import os
import sys
import traceback
import json
from flask import Flask, make_response
from flask import request
from flask import jsonify
from werkzeug.exceptions import HTTPException

# -----------------------------------------------------------------------------
# KAR constants
# -----------------------------------------------------------------------------

# KAR runtime port
if os.getenv("KAR_RUNTIME_PORT") is None:
    raise RuntimeError("KAR_RUNTIME_PORT must be set. Aborting.")
kar_runtime_port = os.getenv("KAR_RUNTIME_PORT")

# KAR app host
kar_app_host = '127.0.0.1'
if os.getenv("KAR_APP_HOST") is not None:
    kar_app_host = os.getenv("KAR_APP_HOST")

# KAR app port
if os.getenv("KAR_APP_PORT") is None:
    raise RuntimeError("KAR_APP_PORT must be set. Aborting.")
kar_app_port = os.getenv("KAR_APP_PORT")

# KAR request timeout in seconds
kar_request_timeout = 10
request_timeout = os.getenv("KAR_REQUEST_TIMEOUT")
if request_timeout is not None:
    request_timeout = int(os.getenv("KAR_REQUEST_TIMEOUT"))
    if request_timeout >= 0:
        kar_request_timeout = request_timeout
kar_request_timeout = aiohttp.ClientTimeout(total=kar_request_timeout)

# Number of retries:
max_retries = 10

# Retry codes:
retry_codes = [503]

# Constants:
actor_type_attribute = "type"
actor_id_attribute = "id"

# -----------------------------------------------------------------------------
# Constant URLs
# -----------------------------------------------------------------------------

# Base URL for the request:
base_url = f'http://localhost:{kar_runtime_port}'

# URL prefix for HTTP requests to sidecar:
sidecar_url_prefix = '/kar/v1'


# -----------------------------------------------------------------------------
# Helper methods
# -----------------------------------------------------------------------------
# TODO: Implement backoff strategy for retries
async def _request(request, api, body=None, headers=None):
    for i in range(max_retries):
        async with request(api, data=body, headers=headers) as response:
            if response.status >= 200 and response.status < 300:
                if response.headers['content-type'] == 'application/json':
                    return await response.json()
                return await response.text()
            if response.status not in retry_codes:
                raise aiohttp.ClientResponseError(
                    response.request_info,
                    response.history,
                    status=response.status,
                    message=response.reason,
                )
    raise RuntimeError("Number of retries exceeded")


# TODO: Implement backoff strategy for retries
async def _actor_request(request, api, body, headers):
    for i in range(max_retries):
        async with request(api, data=body, headers=headers) as response:
            if response.status == 204:
                return await response.text()
            if response.status == 202:
                return response.text()
            if response.status in retry_codes:
                continue
            if response.status != 200:
                raise aiohttp.ClientResponseError(
                    response.request_info,
                    response.history,
                    status=response.status,
                    message=response.reason,
                )
            if response.headers['content-type'] != 'application/kar+json':
                raise RuntimeError(
                    "Response type is not of 'application/kar+json type")
            response = await response.json()
            if "error" in response and response["error"]:
                print(response["stack"], file=sys.stderr)
                # TODO: is this appropriate?
                return
            return response["value"]

    raise RuntimeError("Number of retries exceeded")


async def _fetch(api, options):
    body = None
    if "body" in options:
        body = options["body"]
    headers = {'Content-Type': 'text/plain'}
    if "headers" in options:
        headers = options["headers"]
    async with aiohttp.ClientSession(base_url=base_url,
                                     timeout=kar_request_timeout) as client:
        method = client.get
        if "method" in options:
            method_name = options["method"]
            if method_name == "POST":
                method = client.post
            elif method_name == "PUT":
                method = client.put
            # TODO: add more request types
        return await _request(method, api, body, headers)


async def _post(api, body, headers):
    async with aiohttp.ClientSession(base_url=base_url,
                                     timeout=kar_request_timeout) as client:
        return await _request(client.post, api, body, headers)


async def _get(api, body, headers):
    async with aiohttp.ClientSession(base_url=base_url,
                                     timeout=kar_request_timeout) as client:
        return await _request(client.get, api, body, headers)


async def _delete(api):
    async with aiohttp.ClientSession(base_url=base_url,
                                     timeout=kar_request_timeout) as client:
        return await _request(client.delete, api)


async def _actor_post(api, body, headers):
    async with aiohttp.ClientSession(base_url=base_url,
                                     timeout=kar_request_timeout) as client:
        return await _actor_request(client.post, api, body, headers)


# -----------------------------------------------------------------------------
# Public standalone methods
# -----------------------------------------------------------------------------


#
# KAR invoke
#
# Create a request of the specified `options["method"]` type with an optional
# body given by `options["body"]` and with content type specified in
# `options["headers"]` along with other options. The KAR invoke method requires
# the name of the service and that of the endpoint to be passed in along with
# the above request options.
#
#  Usage for a POST request with Json body:
#        options = {}
#        options["method"] = "POST"
#        options["body"] = json.dumps({"name": "John Doe"})
#        options["headers"] = {'Content-Type': 'application/json'}
#
#        response = await invoke(service_name, endpoint_name, options)
#
async def invoke(service, endpoint, options):
    return await _fetch(
        f'{sidecar_url_prefix}/service/{service}/call/{endpoint}', options)


#
# KAR tell
#
# Create an asynchronous call request that does not expect a response back. The
# method requires the name of the service and that of a service endpoint to be
# passed along with the request body.
#
# The content type for this type of requests is always `application/json`.
#
async def tell(service, endpoint, body):
    return await _post(
        f'{sidecar_url_prefix}/service/{service}/call/{endpoint}', body, {
            'Content-Type': 'application/json',
            'Pragma': 'async'
        })


#
# KAR call
#
# Create an asynchronous call request. The method requires the name of the
# service and that of a service endpoint to be passed along with the request
# body.
#
# The content type for this type of requests is always `application/json`.
#
async def call(service, endpoint, body):
    return await _post(
        f'{sidecar_url_prefix}/service/{service}/call/{endpoint}', body,
        {'Content-Type': 'application/json'})


# -----------------------------------------------------------------------------
# Public actor methods
# -----------------------------------------------------------------------------


#
# Class which represents the generic class of a KAR actor. This class is
# used in two situations: server side and client side.
#
#
# 1. Server-side usage:
# On the server side, the class is used as a base class for a user-created KAR
# actor:
#
#   class MyFirstActor(KarActor):
#       def __init__(self):
#           pass
#
# The KarActor class provides the inheriting class with two attributes: type
# and id which are used by KAR to uniquely identify an actor. To create a valid
# actor, it is actually not required to subclass KarActor. A valid KAR actor is
# a class which has the attributes that the KarActor class defines, currently
# these are represented by `type` and `id`. To be future-proof to changes to
# the KarActor class, we recommend using KarActor as a base class for your
# actors.
#
#
# 2. Client-side:
# On the client side, the KarActor class is used to represent an client-side
# instance of the actor. To create a client-side instance:
#
#   client_side_actor = proxy_actor("MyFirstActor", 123)
#
# The `proxy_actor` is defined below.
#
class KarActor(object):
    def __init__(self):
        self.type = None
        self.id = None


#
# Client-side actor instance. The actor instance on the client side is just an
# instance of the KarActor class which contains two attributes:
#  - type : an attribute which contains the string name of the actor class.
#  - id : an attribute which contains the actor ID which is a unique identifier
#         for the actor. It is the user's responsibility to provide a random,
#         unique ID which does not clash with other instances of this actor
#         type.
# TODO: find a way to provide non-clashing IDs for actors with the same name.
#
# For example for the following Python class:
#
#   class MyFirstActor(KarActor):
#
# create a client-side actor instance:
#
#   client_side_actor = proxy_actor("MyFirstActor", 123)
#
def actor_proxy(actor_type, actor_id):
    actor_proxy = KarActor()
    actor_proxy.type = actor_type
    actor_proxy.id = actor_id
    return actor_proxy


#
# This method is used to remotely call actor methods. The methods can be
# passed arguments and keyword arguments in typical Python style.
#
# To call an actor method several steps are required. This is code which
# typically is written on the client side:
#
#
# 1. Create a client-side actor instance:
#
#   client_side_actor = proxy_actor("MyFirstActor", 123)
#
# This instance can be created anywhere in the user code including in the
# same function which calls the actor method. In this example we will call
# actor creation outside the context of the actor method call itself (see
# below).
#
#
# 2. Call the desired method actor method ensuring `await` is used. This
# requires the actor call to occur in an `async` function.
#
#   async def call_actor_method(client_side_actor):
#       return await actor_call(actor, "method_name", arg1, arg2, kwarg1=value)
#
#
# 3. Calling the async function can happen from anywhere in the user code:
#
#   asyncio.run(call_actor_method(client_side_actor))
#
# Note that `client_side_actor` is passed as argument so create the
# client-side actor before invoking the `call_actor_method`.
#
async def actor_call(*args, **kwargs):
    # Local actor instance which is nothing but a plain KarActor class
    actor = args[0]
    path = args[1]
    body = []
    if len(kwargs) > 0:
        body = {"args": [], "kwargs": {}}
        if len(args) > 2:
            body["args"] = list(args[2:])
        body["kwargs"] = kwargs
    elif len(args) > 2:
        body = args[2:]
    body = json.dumps(body)
    return await _actor_post(
        f"{sidecar_url_prefix}/actor/{actor.type}/{actor.id}/call/{path}",
        body, {'Content-Type': 'application/kar+json'})


#
# Request an actor be explicitely removed from the server side. This method is
# to be called by passing in the client-side actor instance:
#
#  await actor_remove(client_side_actor)
#
# Note this method must be called from a function marked as `async`.
#
async def actor_remove(actor):
    return await _delete(f"{sidecar_url_prefix}/actor/{actor.type}/{actor.id}")


#
# Shutdown the sidecar associated with the current context.
#
async def shutdown():
    return await _post(f"{sidecar_url_prefix}/system/shutdown", None, None)


# -----------------------------------------------------------------------------
# Server actor methods
# -----------------------------------------------------------------------------

_actor_instances = {}

kar_url = "/kar/impl/v1/actor"


def actor_runtime(actors, actor_server=None):
    actor_name_to_type = {}
    for actor_type in actors:
        actor_name_to_type[actor_type.__name__] = actor_type

    if actor_server is None:
        actor_server = Flask(__name__)
        # actor_server.env = "dev"

    @actor_server.errorhandler(Exception)
    def handle_exception(exception):
        # HTTP error (TODO):
        if isinstance(exception, HTTPException):
            return exception

        # non-HTTP error:
        body = {}
        body["error"] = True
        body["stack"] = traceback.format_exc()
        response = make_response(jsonify(body), 200)
        response.headers["Content-Type"] = "application/kar+json"
        return response

    # This method checks if the actor is already active and invokes the
    # activate method if one is provided. This method is automatically invoked
    # by KAR to activate an actor instance.
    @actor_server.route(f"{kar_url}/<string:type>/<int:id>", methods=['GET'])
    def get(type, id):
        # If actor is not present in the list of actor types then return an
        # error to signal that the actor has not been found.
        if type not in actor_name_to_type:
            response = make_response(f"no actor type {type}", 404)
            response.headers["Content-Type"] = "plain/text"
            return response

        # Send response that actor exists.
        if type in _actor_instances and id in _actor_instances[type]:
            response = make_response("existing instance", 200)
            response.headers["Content-Type"] = "plain/text"
            return response

        # Create the actor instance:
        actor_type = actor_name_to_type[type]
        actor_instance = actor_type()
        actor_instance.type = type
        actor_instance.id = id
        if type not in _actor_instances:
            _actor_instances[type] = {}
        _actor_instances[type][id] = actor_instance

        # Call an activate method if one is provided:
        try:
            actor_instance.activate()
            response = make_response("activated", 201)
        except AttributeError:
            response = make_response("created", 201)

        # Send back response:
        response.headers["Content-Type"] = "plain/text"
        return response

    # Method automatically called by KAR to deactivate an actor instance.
    @actor_server.route(f"{kar_url}/<string:type>/<int:id>",
                        methods=['DELETE'])
    def delete(type, id):
        # If actor is not present in the list of actor types then return an
        # error to signal that the actor has not been found.
        if type not in actor_name_to_type:
            response = make_response(f"no actor type {type}", 404)
            response.headers["Content-Type"] = "plain/text"
            return response

        # Check if any instances of this actor exist.
        if type not in _actor_instances:
            response = make_response(f"no instances of actor type {type}", 404)
            response.headers["Content-Type"] = "plain/text"
            return response

        # Check if the actor instance we are looking for exists.
        if id not in _actor_instances[type]:
            response = make_response("no actor with type {type} and id {id}",
                                     404)
            response.headers["Content-Type"] = "plain/text"
            return response

        # Retrieve actor instance
        actor_instance = _actor_instances[type][id]

        # Get deactivate method by name and check if the method is callable.
        # This is an optional method so if the method does not exist do not
        # error.
        try:
            actor_instance.deactivate()
        except AttributeError:
            pass

        # Remove instance from the list of active actor instances:
        del _actor_instances[type][id]

        # Return OK code.
        return ("deleted", 200)

    # Method to call actor methods.
    @actor_server.route(
        f"{kar_url}/<string:type>/<int:id>/<string:session>/<string:method>",
        methods=['POST'])
    async def post(type, id, session, method):
        # Check that the message has JSON type.
        if not request.is_json:
            response = make_response("message data not in JSON format", 404)
            response.headers["Content-Type"] = "plain/text"
            return response

        # Parse input data as JSON if any is provided
        data = None
        if request.data:
            data = request.get_json()

        # If actor is not present in the list of actor types then return an
        # error to signal that the actor has not been found.
        if type not in actor_name_to_type:
            response = make_response(f"no actor type {type}", 404)
            response.headers["Content-Type"] = "plain/text"
            return response

        # If the type exists check that the id exists.
        if type in _actor_instances and id not in _actor_instances[type]:
            response = make_response(f"no actor type {type} with id {id}", 404)
            response.headers["Content-Type"] = "plain/text"
            return response

        # Retrieve actor instance
        actor_instance = _actor_instances[type][id]

        # Fetch the actual actor type.
        actor_type = actor_name_to_type[type]

        # Get actor method by name and check if the method is callable
        try:
            actor_method = getattr(actor_type, method)
            if not callable(actor_method):
                response = make_response(
                    f"no method {method} found for actor ({type}, {id})", 404)
                response.headers["Content-Type"] = "plain/text"
                return response
        except AttributeError:
            response = make_response(
                f"no {method} in actor with type {type} and id {id}", 404)
            response.headers["Content-Type"] = "plain/text"
            return response

        # Call actor method:
        if data:
            if isinstance(data, list):
                if asyncio.iscoroutinefunction(actor_method):
                    result = await actor_method(actor_instance, *data)
                else:
                    result = actor_method(actor_instance, *data)
            else:
                if asyncio.iscoroutinefunction(actor_method):
                    result = await actor_method(actor_instance, *data["args"],
                                                **data["kwargs"])
                else:
                    result = actor_method(actor_instance, *data["args"],
                                          **data["kwargs"])
        else:
            if asyncio.iscoroutinefunction(actor_method):
                result = await actor_method(actor_instance)
            else:
                result = actor_method(actor_instance)

        # If no result was returned, return undefined.
        if result is None:
            response = make_response("undefined", 204)
            response.headers["Content-Type"] = "plain/text"
            return response

        # Return value as JSON and OK code.
        response = make_response(jsonify({"value": result}), 200)
        response.headers["Content-Type"] = "application/kar+json"
        return response

    # Check that the actor type has been registered with KAR.
    # This method is automatically called by KAR to check if the actor type
    # still exists.
    @actor_server.route(f"{kar_url}/<string:type>", methods=['HEAD'])
    def head(type):
        # If actor is not present in the list of actor types then return an
        # error to signal that the actor has not been found.
        if type not in actor_name_to_type:
            response = make_response(f"no actor type {type}", 404)
            response.headers["Content-Type"] = "text/plain"
            return response

        response = make_response("OK", 200)
        response.headers["Content-Type"] = "text/plain"
        return response

    # Health check route.
    @actor_server.route("/kar/impl/v1/system/health", methods=['GET'])
    def health():
        response = make_response("Peachy Keen!", 200)
        response.headers["Content-Type"] = "text/plain"
        return response

    return actor_server
