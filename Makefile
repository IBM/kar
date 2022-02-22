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
KAR_JAVA_REACTIVE_RUNTIME=$(DOCKER_IMAGE_PREFIX)kar-sdk-java-reactive-runtime-11:$(DOCKER_IMAGE_TAG)
KAR_PYTHON_SDK=$(DOCKER_IMAGE_PREFIX)kar-sdk-python-3.8:$(DOCKER_IMAGE_TAG)

KAR_EXAMPLE_JS_YKT=$(DOCKER_IMAGE_PREFIX)kar-examples-js-actors-ykt:$(DOCKER_IMAGE_TAG)
KAR_EXAMPLE_JS_DP=$(DOCKER_IMAGE_PREFIX)kar-examples-js-actors-dp:$(DOCKER_IMAGE_TAG)
KAR_EXAMPLE_JS_HELLO=$(DOCKER_IMAGE_PREFIX)kar-examples-js-service-hello:$(DOCKER_IMAGE_TAG)
KAR_EXAMPLE_JS_EVENTS=$(DOCKER_IMAGE_PREFIX)kar-examples-js-actors-events:$(DOCKER_IMAGE_TAG)
KAR_EXAMPLE_JS_TESTS=$(DOCKER_IMAGE_PREFIX)kar-examples-js-unit-tests:$(DOCKER_IMAGE_TAG)
KAR_EXAMPLE_JAVA_DP=$(DOCKER_IMAGE_PREFIX)kar-examples-java-actors-dp:$(DOCKER_IMAGE_TAG)
KAR_EXAMPLE_JAVA_REACTIVE_DP=$(DOCKER_IMAGE_PREFIX)kar-examples-java-reactive-actors-dp:$(DOCKER_IMAGE_TAG)
KAR_EXAMPLE_JAVA_HELLO=$(DOCKER_IMAGE_PREFIX)kar-examples-java-service-hello:$(DOCKER_IMAGE_TAG)
KAR_EXAMPLE_ACTORS_PYTHON_CONTAINERIZED=$(DOCKER_IMAGE_PREFIX)kar-examples-actors-python-containerized:$(DOCKER_IMAGE_TAG)
KAR_EXAMPLE_ACTORS_PYTHON_CLUSTER=$(DOCKER_IMAGE_PREFIX)kar-examples-actors-python-cluster:$(DOCKER_IMAGE_TAG)
KAR_BENCH_JS_IMAGE=$(DOCKER_IMAGE_PREFIX)kar-bench-js-image:$(DOCKER_IMAGE_TAG)
KAFKA_BENCH=$(DOCKER_IMAGE_PREFIX)kar-kafka-bench:$(DOCKER_IMAGE_TAG)
KAR_HTTP_BENCH_JS_IMAGE=$(DOCKER_IMAGE_PREFIX)kar-http-bench-js-image:$(DOCKER_IMAGE_TAG)

install: cli

cli:
	cd core && go install -ldflags "-X github.com/IBM/kar/core/internal/config.Version=$(KAR_VERSION)" ./...

check-rpc:
	cd core/rpctest && go test

python-sdk:
	cd sdk-python && pip install .


# BUILD BASE images
docker-kar-base:
	cd core && docker build --build-arg KAR_BINARY=kar --build-arg KAR_VERSION=$(KAR_VERSION) -t $(KAR_BASE) .
	cd core && docker build --build-arg KAR_BINARY=kar-injector -t $(KAR_INJECTOR) .


# BUILD SDK images
docker-python-sdk: docker-kar-base
	cd sdk-python && docker build -t $(KAR_PYTHON_SDK) --build-arg KAR_BASE=$(KAR_BASE) .

docker-js-sdk: docker-kar-base
	cd sdk-js && docker build -t $(KAR_JS_SDK) --build-arg KAR_BASE=$(KAR_BASE) .

docker-java-sdk: docker-kar-base
	cd sdk-java && docker build -f Dockerfile.builder -t $(KAR_JAVA_SDK) .
	cd sdk-java && docker build -f Dockerfile.liberty -t $(KAR_JAVA_RUNTIME) --build-arg KAR_BASE=$(KAR_BASE) .
	cd sdk-java && docker build -f Dockerfile.quarkus -t $(KAR_JAVA_REACTIVE_RUNTIME) --build-arg KAR_BASE=$(KAR_BASE) .

dockerBuildCore:
	make docker-python-sdk
	make docker-js-sdk
	make docker-java-sdk


# BUILD EXAMPLE images
docker-python-examples: docker-python-sdk
	cd examples/actors-python && docker build -f Dockerfile.containerized --build-arg PYTHON_RUNTIME=$(KAR_PYTHON_SDK) -t $(KAR_EXAMPLE_ACTORS_PYTHON_CONTAINERIZED) .
	cd examples/actors-python && docker build -f Dockerfile.cluster --build-arg PYTHON_RUNTIME=$(KAR_PYTHON_SDK) -t $(KAR_EXAMPLE_ACTORS_PYTHON_CLUSTER) .

docker-js-examples: docker-js-sdk
	cd examples/actors-dp-js && docker build --build-arg JS_RUNTIME=$(KAR_JS_SDK) -t $(KAR_EXAMPLE_JS_DP) .
	cd examples/actors-events && docker build --build-arg JS_RUNTIME=$(KAR_JS_SDK) -t $(KAR_EXAMPLE_JS_EVENTS) .
	cd examples/actors-ykt && docker build --build-arg JS_RUNTIME=$(KAR_JS_SDK) -t $(KAR_EXAMPLE_JS_YKT) .
	cd examples/service-hello-js && docker build --build-arg JS_RUNTIME=$(KAR_JS_SDK) -t $(KAR_EXAMPLE_JS_HELLO) .
	cd examples/unit-tests && docker build --build-arg JS_RUNTIME=$(KAR_JS_SDK) -t $(KAR_EXAMPLE_JS_TESTS) .

docker-java-examples: docker-java-sdk
	cd examples/actors-dp-java && docker build --build-arg JAVA_BUILDER=$(KAR_JAVA_SDK) --build-arg JAVA_RUNTIME=$(KAR_JAVA_RUNTIME) -t $(KAR_EXAMPLE_JAVA_DP) .
	cd examples/actors-dp-java-reactive && docker build --build-arg JAVA_BUILDER=$(KAR_JAVA_SDK) --build-arg JAVA_RUNTIME=$(KAR_JAVA_REACTIVE_RUNTIME) -t $(KAR_EXAMPLE_JAVA_REACTIVE_DP) .
	cd examples/service-hello-java/server && docker build --build-arg JAVA_BUILDER=$(KAR_JAVA_SDK) --build-arg JAVA_RUNTIME=$(KAR_JAVA_RUNTIME) -t $(KAR_EXAMPLE_JAVA_HELLO) .

dockerBuildExamples:
	make docker-js-examples
	make docker-java-examples
	make docker-python-examples

dockerBuildBenchmarks: docker-js-sdk
	cd benchmark/kar-bench && docker build --build-arg JS_RUNTIME=$(KAR_JS_SDK) -t $(KAR_BENCH_JS_IMAGE) .
	cd benchmark/kafka-bench && docker build -t $(KAFKA_BENCH) .
	cd benchmark/http-bench && docker build --build-arg JS_RUNTIME=$(KAR_JS_SDK) -t $(KAR_HTTP_BENCH_JS_IMAGE) .


# PUSH CORE images
docker-push-base:
	docker push $(KAR_BASE)
	docker push $(KAR_INJECTOR)

docker-push-python-sdk: docker-push-base
	docker push $(KAR_PYTHON_SDK)

docker-push-js-sdk: docker-push-base
	docker push $(KAR_JS_SDK)

docker-push-java-sdk: docker-push-base
	docker push $(KAR_JAVA_SDK)
	docker push $(KAR_JAVA_RUNTIME)
	docker push $(KAR_JAVA_REACTIVE_RUNTIME)

dockerPushCore:
	make docker-push-python-sdk
	make docker-push-js-sdk
	make docker-push-java-sdk


# PUSH EXAMPLES images
docker-push-python-examples: docker-python-examples
	docker push $(KAR_EXAMPLE_ACTORS_PYTHON_CLUSTER)

docker-push-js-examples: docker-js-examples
	docker push $(KAR_EXAMPLE_JS_EVENTS)
	docker push $(KAR_EXAMPLE_JS_DP)
	docker push $(KAR_EXAMPLE_JS_YKT)
	docker push $(KAR_EXAMPLE_JS_HELLO)
	docker push $(KAR_EXAMPLE_JS_TESTS)

docker-push-java-examples: docker-java-examples
	docker push $(KAR_EXAMPLE_JAVA_DP)
	docker push $(KAR_EXAMPLE_JAVA_REACTIVE_DP)
	docker push $(KAR_EXAMPLE_JAVA_HELLO)

dockerPushExamples:
	make docker-push-python-examples
	make docker-push-js-examples
	make docker-push-java-examples

dockerPushBenchmarks:
	docker push $(KAR_BENCH_JS_IMAGE)
	docker push $(KAFKA_BENCH)
	docker push $(KAR_HTTP_BENCH_JS_IMAGE)


# RUN containerized examples
docker-run-containerized-python-examples: docker-python-examples
	docker run --network kar-bus --add-host=host.docker.internal:host-gateway $(KAR_EXAMPLE_ACTORS_PYTHON_CONTAINERIZED)


# Build and push ALL docker-related images
docker-python:
	make docker-python-examples
	make docker-push-python-sdk
	make docker-push-python-examples

docker-js:
	make docker-js-examples
	make docker-push-js-sdk
	make docker-push-js-examples

docker-java:
	make docker-java-examples
	make docker-push-java-sdk
	make docker-push-java-examples

docker:
	make dockerBuildCore
	make dockerBuildExamples
	make dockerBuildBenchmarks
	make dockerPushCore
	make dockerPushExamples
	make dockerPushBenchmarks


# Build core images
dockerBuild:
	make dockerBuildCore
	make dockerBuildExamples


# Install Java
installJavaSDK:
	cd sdk-java && mvn install


# Swagger
swagger-gen:
	cd core && swagger generate spec -o ../docs/api/swagger.yaml
	cd core && swagger generate spec -o ../docs/api/swagger.json

swagger-serve:
	swagger serve docs/api/swagger.yaml
