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

DOCKER_REGISTRY ?= localhost:5000
DOCKER_NAMESPACE ?= kar
DOCKER_IMAGE_PREFIX ?= $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/
DOCKER_IMAGE_TAG ?= latest
KAR_JS_SDK=$(DOCKER_IMAGE_PREFIX)kar-sdk-nodejs-v18:$(DOCKER_IMAGE_TAG)

docker:
	make dockerBuild
	make dockerPush

dockerBuild:
	docker build --build-arg JS_RUNTIME=$(KAR_JS_SDK) -t $(DOCKER_IMAGE_PREFIX)kar-examples-stock-processor .
	KAFKA_BROKERS='{{env:KAFKA_BROKERS}}' kamel local create --image $(DOCKER_IMAGE_PREFIX)kar-examples-stock-source input.yaml CloudEventProcessor.java
	KAFKA_BROKERS='{{env:KAFKA_BROKERS}}' SLACK_WEBHOOK='{{env:SLACK_WEBHOOK}}' kamel local create --image $(DOCKER_IMAGE_PREFIX)kar-examples-stock-sink output.yaml CloudEventProcessor.java

dockerPush:
	docker push $(DOCKER_IMAGE_PREFIX)kar-examples-stock-processor
	docker push $(DOCKER_IMAGE_PREFIX)kar-examples-stock-source
	docker push $(DOCKER_IMAGE_PREFIX)kar-examples-stock-sink
