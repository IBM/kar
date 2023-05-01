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

# Build KAR
FROM golang:1.19 as builder

WORKDIR /kar/core

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY pkg pkg
COPY internal internal
COPY cmd cmd

ARG KAR_VERSION=unofficial

RUN CGO_ENABLED=0 go install -ldflags "-X github.com/IBM/kar/core/internal/config.Version=$KAR_VERSION" ./...

# Create image containing the sidecar
FROM alpine:3.17
ARG KAR_BINARY

COPY --from=builder /go/bin/$KAR_BINARY /kar/bin/$KAR_BINARY
