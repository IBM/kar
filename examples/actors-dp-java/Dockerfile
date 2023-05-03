#
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
#

ARG JAVA_BUILDER=localhost:5000/kar/kar-sdk-java-builder-11
ARG JAVA_RUNTIME=localhost:5000/kar/kar-sdk-java-runtime-11

#
# Build the artifacts using the builder image
#
FROM $JAVA_BUILDER as builder

WORKDIR /kar/app

COPY pom.xml ./
RUN mvn -q dependency:go-offline

COPY src src
RUN mvn -q package

#
# Copy the artifacts to the runtime image
#
FROM $JAVA_RUNTIME
COPY --from=builder --chown=default:root /kar/app/target/philosophers-liberty.war /opt/ol/wlp/usr/servers/defaultServer/apps
COPY --from=builder --chown=default:root /kar/app/src/main/liberty/config/server.xml /opt/ol/wlp/usr/servers/defaultServer

#
# Steps below based on: https://github.com/OpenLiberty/ci.docker#building-an-application-image
#

RUN features.sh

RUN configure.sh
