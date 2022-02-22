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

from flask import Flask
from flask import request
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

# Setup Flask
app = Flask(__name__)
app.env = "dev"


@app.route('/helloText', methods=['POST'])
def post_hello_text():
    if request.method == 'POST':
        data = request.get_data(as_text=True)
    greetings_message = "Hello " + data
    print(greetings_message, file=sys.stderr)
    return greetings_message


@app.route('/helloJson', methods=['POST'])
def post_hello_json():
    if request.method == 'POST':
        if request.is_json:
            data = request.get_json()
        else:
            print("Received request is not in JSON format.", file=sys.stderr)
    greetings_message = "Hello " + data["name"]
    print(greetings_message, file=sys.stderr)
    return {"greetings": greetings_message}


@app.route('/health', methods=['GET'])
def health_check():
    greetings_message = "I am healthy"
    print(greetings_message, file=sys.stderr)
    return greetings_message


if __name__ == '__main__':
    print(kar_app_host, kar_app_port)
    app.run(host=kar_app_host, port=kar_app_port, debug=True)
