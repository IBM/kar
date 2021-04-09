#
# Copyright IBM Corporation 2020,2021
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

# Resolve dependencies
FROM golang:1.15.11-alpine AS build
RUN apk add git
RUN go get -v "github.com/Shopify/sarama"
RUN go get -v "github.com/prometheus/common/log"

# Build executable
FROM golang:1.15.11-alpine AS builder
COPY --from=build /go /go
ENV CGO_ENABLED=0
WORKDIR /src
COPY main.go .
RUN go build -o consumer .
CMD ["/src/consumer"]