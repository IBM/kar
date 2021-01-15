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

DOCKER_REGISTRY ?= us.icr.io
DOCKER_NAMESPACE ?= research/kar-dev
DOCKER_IMAGE_PREFIX ?= $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/
DOCKER_IMAGE_TAG ?= latest

KAR_JS_SDK=$(DOCKER_IMAGE_PREFIX)sdk-nodejs-v12:$(DOCKER_IMAGE_TAG)

dockerDev:
	DOCKER_IMAGE_PREFIX=localhost:5000/ make docker

docker:
	make dockerBuild
	make dockerPush

dockerBuild:
	docker build --build-arg JS_RUNTIME=$(KAR_JS_SDK) -t $(DOCKER_IMAGE_PREFIX)examples/stock-processor .
	KAFKA_BROKERS='{{env:KAFKA_BROKERS}}' kamel local create --image $(DOCKER_IMAGE_PREFIX)examples/stock-source input.yaml ../../scripts/kamel/CloudEventProcessor.java 
	KAFKA_BROKERS='{{env:KAFKA_BROKERS}}' SLACK_WEBHOOK='{{env:SLACK_WEBHOOK}}' kamel local create --image $(DOCKER_IMAGE_PREFIX)examples/stock-sink output.yaml ../../scripts/kamel/CloudEventProcessor.java 

dockerPush:
	docker push $(DOCKER_IMAGE_PREFIX)examples/stock-processor
	docker push $(DOCKER_IMAGE_PREFIX)examples/stock-source
	docker push $(DOCKER_IMAGE_PREFIX)examples/stock-sink
