DOCKER_REGISTRY ?= localhost:5000
DOCKER_NAMESPACE ?= kar
DOCKER_IMAGE_PREFIX ?= $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/
DOCKER_IMAGE_TAG ?= latest

KAFKA_BENCH=$(DOCKER_IMAGE_PREFIX)kar-kafka-bench:$(DOCKER_IMAGE_TAG)

dockerBuild:
	docker build -t $(KAFKA_BENCH) .

dockerPush:
	docker push $(KAFKA_BENCH)

docker:
	make dockerBuild
	make dockerPush

build:
	go install ./...
