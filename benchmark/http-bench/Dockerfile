#
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
#

FROM node:18-alpine

WORKDIR /kar/app

# Default to running actor server
ENV MAIN=server.js

# Copy dependencies first; they change rarely
COPY package.json package-lock.json ./
RUN npm ci --omit=dev

# Copy in application source
COPY *.js ./

CMD cd /kar/app && node $MAIN
