DOCKER_REGISTRY ?= us.icr.io
DOCKER_NAMESPACE ?= research/kar-dev
DOCKER_IMAGE_PREFIX ?= $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/
DOCKER_IMAGE_TAG ?= latest

KAR_BASE=$(DOCKER_IMAGE_PREFIX)kar:$(DOCKER_IMAGE_TAG)
KAR_INJECTOR=$(DOCKER_IMAGE_PREFIX)kar-injector:$(DOCKER_IMAGE_TAG)
KAR_JS_SDK=$(DOCKER_IMAGE_PREFIX)sdk-nodejs-v12:$(DOCKER_IMAGE_TAG)
KAR_JS_EXAMPLES=$(DOCKER_IMAGE_PREFIX)examples-js:$(DOCKER_IMAGE_TAG)

all: install

install:
	go install ./...

docker:
	docker build -f build/Dockerfile --build-arg KAR_BINARY=kar -t $(KAR_BASE) .
	docker build -f build/Dockerfile --build-arg KAR_BINARY=kar-injector -t $(KAR_INJECTOR) .
	docker build -t $(KAR_JS_SDK) sdk/js
	docker build -f examples/docker/Dockerfile_js -t $(KAR_JS_EXAMPLES) --build-arg KAR_BASE=$(KAR_BASE) --build-arg SDK_BASE=$(KAR_JS_SDK) examples

dockerPush: docker
	docker push $(KAR_BASE)
	docker push $(KAR_INJECTOR)
	docker push $(KAR_JS_SDK)
	docker push $(KAR_JS_EXAMPLES)

kindPush: docker
	kind load docker-image $(KAR_BASE)
	kind load docker-image $(KAR_INJECTOR)
	kind load docker-image $(KAR_JS_SDK)
	kind load docker-image $(KAR_JS_EXAMPLES)

kindPushDev:
	DOCKER_IMAGE_PREFIX= DOCKER_IMAGE_TAG=dev make kindPush

swagger-gen:
	swagger generate spec -o docs/api/swagger.yaml
	swagger generate spec -o docs/api/swagger.json

swagger-serve:
	swagger serve docs/api/swagger.yaml

.PHONY: docker
