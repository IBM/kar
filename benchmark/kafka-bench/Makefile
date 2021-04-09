DOCKER_REGISTRY ?= localhost:5000
DOCKER_NAMESPACE ?= kar
DOCKER_IMAGE_PREFIX ?= $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/
DOCKER_IMAGE_TAG ?= latest

KAFKA_BENCH_CONSUMER=$(DOCKER_IMAGE_PREFIX)kar-kafka-bench-consumer:$(DOCKER_IMAGE_TAG)
KAFKA_BENCH_PRODUCER=$(DOCKER_IMAGE_PREFIX)kar-kafka-bench-producer:$(DOCKER_IMAGE_TAG)

dockerBuild:
	cd consumer && docker build -t $(KAFKA_BENCH_CONSUMER) .
	cd producer && docker build -t $(KAFKA_BENCH_PRODUCER) .

dockerPush:
	docker push $(KAFKA_BENCH_CONSUMER)
	docker push $(KAFKA_BENCH_PRODUCER)

docker:
	make dockerBuild
	make dockerPush

build:
	cd producer; go build
	cd consumer; go build
