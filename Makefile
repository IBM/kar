DOCKER_IMAGE ?= kar
DOCKER_TAG ?= latest

all: build

build:
	go install ./...

docker:
	docker build -f docker/Dockerfile -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

.PHONY: docker
