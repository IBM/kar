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

KAR_VERSION ?= unofficial
KAR_JS_SDK=$(DOCKER_IMAGE_PREFIX)kar-sdk-nodejs-v12:$(DOCKER_IMAGE_TAG)

KAR_BENCH_JS_IMAGE=$(DOCKER_IMAGE_PREFIX)kar-bench-js-image:$(DOCKER_IMAGE_TAG)

dockerBuild:
	docker build --build-arg JS_RUNTIME=$(KAR_JS_SDK) -t $(KAR_BENCH_JS_IMAGE) .

dockerPush:
	docker push $(KAR_BENCH_JS_IMAGE)

docker:
	make dockerBuild
	make dockerPush
