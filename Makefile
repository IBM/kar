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
	s2i build examples/actors-events $(KAR_JS_SDK) $(KAR_EXAMPLE_JS_EVENTS) --copy
	s2i build examples/actors-ykt $(KAR_JS_SDK) $(KAR_EXAMPLE_JS_YKT) --copy
	s2i build examples/helloWorld $(KAR_JS_SDK) $(KAR_EXAMPLE_JS_HELLO) --copy
	s2i build examples/stockPriceEvents $(KAR_JS_SDK) $(KAR_EXAMPLE_JS_STOCK) --copy
	s2i build examples/unit-tests $(KAR_JS_SDK) $(KAR_EXAMPLE_JS_TESTS) --copy
	s2i build examples/java/actors $(KAR_JAVA_SDK) $(KAR_EXAMPLE_JAVA_ACTORS) --runtime-image $(KAR_JAVA_RUNTIME) --assemble-runtime-user root --runtime-artifact /kar/app/kar-actor-example/target/kar-actor-example.war:opt/ol/wlp/usr/servers/defaultServer/apps --runtime-artifact /kar/app/kar-actor-example/src/main/liberty/config/server.xml:opt/ol/wlp/usr/servers/defaultServer --copy

dockerPushCore:
	docker push $(KAR_BASE)
	docker push $(KAR_INJECTOR)
	docker push $(KAR_JS_SDK)
	docker push $(KAR_JAVA_SDK)

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
