DOCKER_REGISTRY ?= localhost:5000
DOCKER_NAMESPACE ?= kar
DOCKER_IMAGE_PREFIX ?= $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/
DOCKER_IMAGE_TAG ?= latest

KAR_JS_SDK=$(DOCKER_IMAGE_PREFIX)kar-sdk-nodejs-v12:$(DOCKER_IMAGE_TAG)

KAR_HTTP_BENCH_JS_IMAGE=$(DOCKER_IMAGE_PREFIX)kar-http-bench-js-image:$(DOCKER_IMAGE_TAG)

dockerBuild:
	docker build --build-arg JS_RUNTIME=$(KAR_JS_SDK) -t $(KAR_HTTP_BENCH_JS_IMAGE) .

dockerPush:
	docker push $(KAR_HTTP_BENCH_JS_IMAGE)

docker:
	make dockerBuild
	make dockerPush
