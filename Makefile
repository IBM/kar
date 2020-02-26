DOCKER_REGISTRY ?= us.icr.io
DOCKER_NAMESPACE ?= kar-dev
DOCKER_IMAGE ?= $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/kar
DOCKER_IMAGE_TAG ?= latest

all: build

build:
	go install ./...

docker:
	docker build -f docker/Dockerfile -t $(DOCKER_IMAGE):$(DOCKER_IMAGE_TAG) .

dockerPush: docker
	docker push $(DOCKER_IMAGE):$(DOCKER_IMAGE_TAG)

.PHONY: docker
