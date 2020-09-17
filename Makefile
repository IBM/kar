DOCKER_REGISTRY ?= us.icr.io
DOCKER_NAMESPACE ?= research/kar-dev
DOCKER_IMAGE_PREFIX ?= $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/
DOCKER_IMAGE_TAG ?= latest

KAR_BASE=$(DOCKER_IMAGE_PREFIX)kar:$(DOCKER_IMAGE_TAG)
KAR_INJECTOR=$(DOCKER_IMAGE_PREFIX)kar-injector:$(DOCKER_IMAGE_TAG)
KAR_JS_SDK=$(DOCKER_IMAGE_PREFIX)sdk-nodejs-v12:$(DOCKER_IMAGE_TAG)
KAR_JAVA_SDK=$(DOCKER_IMAGE_PREFIX)sdk-java-builder-11:$(DOCKER_IMAGE_TAG)
KAR_JAVA_RUNTIME=$(DOCKER_IMAGE_PREFIX)sdk-java-runtime-11:$(DOCKER_IMAGE_TAG)

KAR_EXAMPLE_JS_YKT=$(DOCKER_IMAGE_PREFIX)examples/js/actors-ykt:$(DOCKER_IMAGE_TAG)
KAR_EXAMPLE_JS_DP=$(DOCKER_IMAGE_PREFIX)examples/js/actors-dp:$(DOCKER_IMAGE_TAG)
KAR_EXAMPLE_JS_HELLO=$(DOCKER_IMAGE_PREFIX)examples/js/hello-world:$(DOCKER_IMAGE_TAG)
KAR_EXAMPLE_JS_EVENTS=$(DOCKER_IMAGE_PREFIX)examples/js/actors-events:$(DOCKER_IMAGE_TAG)
KAR_EXAMPLE_JS_STOCK=$(DOCKER_IMAGE_PREFIX)examples/js/stock-prices:$(DOCKER_IMAGE_TAG)
KAR_EXAMPLE_JS_TESTS=$(DOCKER_IMAGE_PREFIX)examples/js/unit-tests:$(DOCKER_IMAGE_TAG)
KAR_EXAMPLE_JAVA_ACTORS=$(DOCKER_IMAGE_PREFIX)examples/java/actors:$(DOCKER_IMAGE_TAG)

all: install

install:
	cd core && go install ./...

dockerCore:
	cd core && docker build --build-arg KAR_BINARY=kar -t $(KAR_BASE) .
	cd core && docker build --build-arg KAR_BINARY=kar-injector -t $(KAR_INJECTOR) .
	cd sdk-js && docker build -t $(KAR_JS_SDK) --build-arg KAR_BASE=$(KAR_BASE) .
	cd sdk-java && docker build -f Dockerfile.builder -t $(KAR_JAVA_SDK) .
	cd sdk-java && docker build -f Dockerfile.runtime -t $(KAR_JAVA_RUNTIME) --build-arg KAR_BASE=$(KAR_BASE) .

dockerExamples:
	cd examples/actors-dp-js && docker build --build-arg JS_RUNTIME=$(KAR_JS_SDK) -t $(KAR_EXAMPLE_JS_DP) .
	cd examples/actors-events && docker build --build-arg JS_RUNTIME=$(KAR_JS_SDK) -t $(KAR_EXAMPLE_JS_EVENTS) .
	cd examples/actors-ykt && docker build --build-arg JS_RUNTIME=$(KAR_JS_SDK) -t $(KAR_EXAMPLE_JS_YKT) .
	cd examples/helloWorld && docker build --build-arg JS_RUNTIME=$(KAR_JS_SDK) -t $(KAR_EXAMPLE_JS_HELLO) .
	cd examples/stockPriceEvents && docker build --build-arg JS_RUNTIME=$(KAR_JS_SDK) -t $(KAR_EXAMPLE_JS_STOCK) .
	cd examples/unit-tests && docker build --build-arg JS_RUNTIME=$(KAR_JS_SDK) -t $(KAR_EXAMPLE_JS_TESTS) . 
	cd examples/java/actors && docker build --build-arg JAVA_BUILDER=$(KAR_JAVA_SDK) --build-arg JAVA_RUNTIME=$(KAR_JAVA_RUNTIME) -t $(KAR_EXAMPLE_JAVA_ACTORS) .

dockerPushCore:
	docker push $(KAR_BASE)
	docker push $(KAR_INJECTOR)
	docker push $(KAR_JS_SDK)
	docker push $(KAR_JAVA_SDK)
	docker push $(KAR_JAVA_RUNTIME)

dockerPushExamples:
	docker push $(KAR_EXAMPLE_JS_EVENTS)
	docker push $(KAR_EXAMPLE_JS_YKT)
	docker push $(KAR_EXAMPLE_JS_HELLO)
	docker push $(KAR_EXAMPLE_JS_STOCK)
	docker push $(KAR_EXAMPLE_JS_TESTS)
	docker push $(KAR_EXAMPLE_JAVA_ACTORS)

dockerBuildAndPush:
	make dockerCore
	make dockerExamples
	make dockerPushCore
	make dockerPushExamples

dockerDev:
	DOCKER_IMAGE_PREFIX=localhost:5000/ make dockerCore dockerExamples
	DOCKER_IMAGE_PREFIX=localhost:5000/ make dockerPushCore dockerPushExamples

installJavaSDK:
	cd sdk-java && mvn install

swagger-gen:
	cd core && swagger generate spec -o ../docs/api/swagger.yaml
	cd core && swagger generate spec -o ../docs/api/swagger.json

swagger-serve:
	swagger serve docs/api/swagger.yaml

.PHONY: docker
