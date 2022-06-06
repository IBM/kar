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

ARG KAR_BASE=localhost:5000/kar/kar-sidecar
ARG PYTHON_VERSION=3.8

FROM $KAR_BASE as kar

FROM python:${PYTHON_VERSION}-alpine

# Setup the working directory:
RUN mkdir /python_kar
WORKDIR /python_kar

# Copy Python SDK:
COPY kar kar
COPY setup.py setup.py

# Copy KAR executable:
RUN mkdir /python_kar/bin
COPY --from=kar /kar/bin/kar /python_kar/bin/kar

# Install Python SDK:
RUN pip install -e .
