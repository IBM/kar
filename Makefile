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

DOCKER_REGISTRY ?= localhost:5000
DOCKER_NAMESPACE ?= kar
DOCKER_IMAGE_PREFIX ?= $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/
DOCKER_IMAGE_TAG ?= latest

KAR_VERSION ?= unofficial

KAR_BASE=$(DOCKER_IMAGE_PREFIX)kar-sidecar:$(DOCKER_IMAGE_TAG)
KAR_INJECTOR=$(DOCKER_IMAGE_PREFIX)kar-injector:$(DOCKER_IMAGE_TAG)
KAR_JS_SDK=$(DOCKER_IMAGE_PREFIX)kar-sdk-nodejs-v12:$(DOCKER_IMAGE_TAG)
KAR_JAVA_SDK=$(DOCKER_IMAGE_PREFIX)kar-sdk-java-builder-11:$(DOCKER_IMAGE_TAG)
KAR_JAVA_RUNTIME=$(DOCKER_IMAGE_PREFIX)kar-sdk-java-runtime-11:$(DOCKER_IMAGE_TAG)

KAR_EXAMPLE_JS_YKT=$(DOCKER_IMAGE_PREFIX)kar-examples-js-actors-ykt:$(DOCKER_IMAGE_TAG)
KAR_EXAMPLE_JS_DP=$(DOCKER_IMAGE_PREFIX)kar-examples-js-actors-dp:$(DOCKER_IMAGE_TAG)
KAR_EXAMPLE_JS_HELLO=$(DOCKER_IMAGE_PREFIX)kar-examples-js-service-hello:$(DOCKER_IMAGE_TAG)
KAR_EXAMPLE_JS_EVENTS=$(DOCKER_IMAGE_PREFIX)kar-examples-js-actors-events:$(DOCKER_IMAGE_TAG)
KAR_EXAMPLE_JS_TESTS=$(DOCKER_IMAGE_PREFIX)kar-examples-js-unit-tests:$(DOCKER_IMAGE_TAG)
KAR_EXAMPLE_JAVA_DP=$(DOCKER_IMAGE_PREFIX)kar-examples-java-actors-dp:$(DOCKER_IMAGE_TAG)
KAR_EXAMPLE_JAVA_HELLO=$(DOCKER_IMAGE_PREFIX)kar-examples-java-service-hello:$(DOCKER_IMAGE_TAG)
KAR_BENCH_JS_IMAGE=$(DOCKER_IMAGE_PREFIX)kar-bench-js-image:$(DOCKER_IMAGE_TAG)
KAFKA_BENCH_CONSUMER=$(DOCKER_IMAGE_PREFIX)kar-kafka-bench-consumer:$(DOCKER_IMAGE_TAG)
KAFKA_BENCH_PRODUCER=$(DOCKER_IMAGE_PREFIX)kar-kafka-bench-producer:$(DOCKER_IMAGE_TAG)
KAR_HTTP_BENCH_JS_IMAGE=$(DOCKER_IMAGE_PREFIX)kar-http-bench-js-image:$(DOCKER_IMAGE_TAG)

install: cli

cli:
	cd core && go install -ldflags "-X github.com/IBM/kar.git/core/internal/config.Version=$(KAR_VERSION)" ./...

dockerBuildCore:
	cd core && docker build --build-arg KAR_BINARY=kar --build-arg KAR_VERSION=$(KAR_VERSION) -t $(KAR_BASE) .
	cd core && docker build --build-arg KAR_BINARY=kar-injector -t $(KAR_INJECTOR) .
	cd sdk-js && docker build -t $(KAR_JS_SDK) --build-arg KAR_BASE=$(KAR_BASE) .
	cd sdk-java && docker build -f Dockerfile.builder -t $(KAR_JAVA_SDK) .
	cd sdk-java && docker build -f Dockerfile.runtime -t $(KAR_JAVA_RUNTIME) --build-arg KAR_BASE=$(KAR_BASE) .

dockerBuildExamples:
	cd examples/actors-dp-js && docker build --build-arg JS_RUNTIME=$(KAR_JS_SDK) -t $(KAR_EXAMPLE_JS_DP) .
	cd examples/actors-events && docker build --build-arg JS_RUNTIME=$(KAR_JS_SDK) -t $(KAR_EXAMPLE_JS_EVENTS) .
	cd examples/actors-ykt && docker build --build-arg JS_RUNTIME=$(KAR_JS_SDK) -t $(KAR_EXAMPLE_JS_YKT) .
	cd examples/service-hello-js && docker build --build-arg JS_RUNTIME=$(KAR_JS_SDK) -t $(KAR_EXAMPLE_JS_HELLO) .
	cd examples/unit-tests && docker build --build-arg JS_RUNTIME=$(KAR_JS_SDK) -t $(KAR_EXAMPLE_JS_TESTS) . 
	cd examples/actors-dp-java && docker build --build-arg JAVA_BUILDER=$(KAR_JAVA_SDK) --build-arg JAVA_RUNTIME=$(KAR_JAVA_RUNTIME) -t $(KAR_EXAMPLE_JAVA_DP) .
	cd examples/service-hello-java/server && docker build --build-arg JAVA_BUILDER=$(KAR_JAVA_SDK) --build-arg JAVA_RUNTIME=$(KAR_JAVA_RUNTIME) -t $(KAR_EXAMPLE_JAVA_HELLO) .

dockerBuildBenchmarks:
	cd benchmark/kar-bench && docker build --build-arg JS_RUNTIME=$(KAR_JS_SDK) -t $(KAR_BENCH_JS_IMAGE) .
# DISABLED DUE TO https://github.com/IBM/kar/issues/118
#	cd benchmark/kafka-bench/consumer && docker build -t $(KAFKA_BENCH_CONSUMER) .
#	cd benchmark/kafka-bench/producer && docker build -t $(KAFKA_BENCH_PRODUCER) .
	cd benchmark/http-bench && docker build --build-arg JS_RUNTIME=$(KAR_JS_SDK) -t $(KAR_HTTP_BENCH_JS_IMAGE) .

dockerPushCore:
	docker push $(KAR_BASE)
	docker push $(KAR_INJECTOR)
	docker push $(KAR_JS_SDK)
	docker push $(KAR_JAVA_SDK)
	docker push $(KAR_JAVA_RUNTIME)

dockerPushExamples:
	docker push $(KAR_EXAMPLE_JS_EVENTS)
	docker push $(KAR_EXAMPLE_JS_DP)
	docker push $(KAR_EXAMPLE_JS_YKT)
	docker push $(KAR_EXAMPLE_JS_HELLO)
	docker push $(KAR_EXAMPLE_JS_TESTS)
	docker push $(KAR_EXAMPLE_JAVA_DP)
	docker push $(KAR_EXAMPLE_JAVA_HELLO)

dockerPushBenchmarks:
	docker push $(KAR_BENCH_JS_IMAGE)
# DISABLED DUE TO https://github.com/IBM/kar/issues/118
#	docker push $(KAFKA_BENCH_CONSUMER)
#	docker push $(KAFKA_BENCH_PRODUCER)
	docker push $(KAR_HTTP_BENCH_JS_IMAGE)

docker:
	make dockerBuildCore
	make dockerBuildExamples
	make dockerBuildBenchmarks
	make dockerPushCore
	make dockerPushExamples
	make dockerPushBenchmarks

dockerBuild:
	make dockerBuildCore
	make dockerBuildExamples

installJavaSDK:
	cd sdk-java && mvn install

swagger-gen:
	cd core && swagger generate spec -o ../docs/api/swagger.yaml
	cd core && swagger generate spec -o ../docs/api/swagger.json

swagger-serve:
	swagger serve docs/api/swagger.yaml
