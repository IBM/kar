DOCKER_REGISTRY ?= us.icr.io
DOCKER_NAMESPACE ?= kar-dev
DOCKER_IMAGE_PREFIX ?= $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/
DOCKER_IMAGE_TAG ?= latest

KAR_JS_SDK=$(DOCKER_IMAGE_PREFIX)sdk-nodejs-v12:$(DOCKER_IMAGE_TAG)

all: install

install:
	go install ./...

docker:
	docker build -f build/Dockerfile --build-arg KAR_BINARY=kar -t $(DOCKER_IMAGE_PREFIX)kar:$(DOCKER_IMAGE_TAG) .
	docker build -f build/Dockerfile --build-arg KAR_BINARY=kar-injector -t $(DOCKER_IMAGE_PREFIX)kar-injector:$(DOCKER_IMAGE_TAG) .
	docker build -t $(KAR_JS_SDK) sdk/js
	docker build -t $(DOCKER_IMAGE_PREFIX)example-hello-world:$(DOCKER_IMAGE_TAG) --build-arg SDK_BASE=$(KAR_JS_SDK) examples/helloWorld
	docker build -t $(DOCKER_IMAGE_PREFIX)example-unit-tests:$(DOCKER_IMAGE_TAG) --build-arg SDK_BASE=$(KAR_JS_SDK) examples/unit-tests
	docker build -t $(DOCKER_IMAGE_PREFIX)example-ykt:$(DOCKER_IMAGE_TAG) --build-arg SDK_BASE=$(KAR_JS_SDK) examples/actors-ykt

dockerPush: docker
	docker push $(DOCKER_IMAGE_PREFIX)kar:$(DOCKER_IMAGE_TAG)
	docker push $(DOCKER_IMAGE_PREFIX)kar-injector:$(DOCKER_IMAGE_TAG)
	docker push $(KAR_JS_SDK)
	docker push $(DOCKER_IMAGE_PREFIX)example-hello-world:$(DOCKER_IMAGE_TAG)
	docker push $(DOCKER_IMAGE_PREFIX)example-unit-tests:$(DOCKER_IMAGE_TAG)
	docker push $(DOCKER_IMAGE_PREFIX)example-ykt:$(DOCKER_IMAGE_TAG)

kindPush: docker
	kind load docker-image $(DOCKER_IMAGE_PREFIX)kar:$(DOCKER_IMAGE_TAG)
	kind load docker-image $(DOCKER_IMAGE_PREFIX)kar-injector:$(DOCKER_IMAGE_TAG)
	kind load docker-image $(KAR_JS_SDK)
	kind load docker-image $(DOCKER_IMAGE_PREFIX)example-hello-world:$(DOCKER_IMAGE_TAG)
	kind load docker-image $(DOCKER_IMAGE_PREFIX)example-unit-tests:$(DOCKER_IMAGE_TAG)
	kind load docker-image $(DOCKER_IMAGE_PREFIX)example-ykt:$(DOCKER_IMAGE_TAG)

kindPushDev:
	DOCKER_IMAGE_PREFIX= DOCKER_IMAGE_TAG=dev make kindPush

swagger-gen:
	swagger generate spec -o docs/swagger.yaml
	swagger generate spec -o docs/swagger.json

swagger-serve:
	swagger serve docs/swagger.yaml

.PHONY: docker
