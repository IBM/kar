DOCKER_REGISTRY ?= us.icr.io
DOCKER_NAMESPACE ?= kar-dev
DOCKER_IMAGE_PREFIX ?= $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/
DOCKER_IMAGE_TAG ?= latest

all: install

install:
	go install ./...

kar:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build cmd/kar/kar.go

docker: kar
	docker build -f build/Dockerfile -t $(DOCKER_IMAGE_PREFIX)kar:$(DOCKER_IMAGE_TAG) .
	docker build -f examples/incr/Dockerfile -t $(DOCKER_IMAGE_PREFIX)sample-incr:$(DOCKER_IMAGE_TAG) examples/incr

dockerPush: docker
	docker push $(DOCKER_IMAGE_PREFIX)kar:$(DOCKER_IMAGE_TAG)
	docker push $(DOCKER_IMAGE_PREFIX)sample-incr:$(DOCKER_IMAGE_TAG)

.PHONY: docker
