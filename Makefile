DOCKER_REGISTRY ?= us.icr.io
DOCKER_NAMESPACE ?= kar-dev
DOCKER_IMAGE_PREFIX ?= $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/
DOCKER_IMAGE_TAG ?= latest

all: build

build:
	go install ./...

docker:
	docker build -f docker/Dockerfile -t $(DOCKER_IMAGE_PREFIX)kar:$(DOCKER_IMAGE_TAG) .
	docker build -f samples/incr/Dockerfile -t $(DOCKER_IMAGE_PREFIX)sample-incr:$(DOCKER_IMAGE_TAG) samples/incr

dockerPush: docker
	docker push $(DOCKER_IMAGE_PREFIX)kar:$(DOCKER_IMAGE_TAG)
	docker push $(DOCKER_IMAGE_PREFIX)sample-incr:$(DOCKER_IMAGE_TAG)

.PHONY: docker
