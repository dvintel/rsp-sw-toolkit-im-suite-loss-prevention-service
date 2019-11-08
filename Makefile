
SERVICE_NAME ?= loss-prevention
PROJECT_NAME ?= loss-prevention-service

BUILDER_IMAGE ?= gocv-openvino-builder
RUNTIME_IMAGE ?= openvino-runtime

# The default flags to use when calling submakes
GNUMAKEFLAGS = --no-print-directory

GIT_SHA = $(shell git rev-parse HEAD)

GO_FILES = $(shell find . -type f -name '*.go')
RES_FILES = $(shell find res/ -type f)

PROXY_ARGS =	--build-arg http_proxy=$(http_proxy) \
				--build-arg https_proxy=$(https_proxy) \
				--build-arg no_proxy=$(no_proxy) \
				--build-arg HTTP_PROXY=$(HTTP_PROXY) \
				--build-arg HTTPS_PROXY=$(HTTPS_PROXY) \
				--build-arg NO_PROXY=$(NO_PROXY)

EXTRA_BUILD_ARGS ?=

touch_target_file = mkdir -p $(@D) && touch $@

trap_ctrl_c = trap 'exit 0' INT;

compose = docker-compose

log = docker-compose logs $1 $2 2>&1

.PHONY: build clean iterate iterate-d tail start stop rm deploy kill down fmt ps delete-all-recordings shell/*

default: build

build: build/docker

clean:
	rm -rf build/*

build/openvino-builder: go.mod Dockerfile.builder
	docker build \
		--build-arg GIT_TOKEN=$(GIT_TOKEN) \
		$(PROXY_ARGS) \
		$(EXTRA_BUILD_ARGS) \
		--build-arg LOCAL_USER=$$(id -u $$(logname)) \
		-f Dockerfile.builder \
		--label "git_sha=$(GIT_SHA)" \
		-t rsp/$(BUILDER_IMAGE):dev \
		.
	@$(touch_target_file)

$(PROJECT_NAME): build/openvino-builder Makefile build.sh $(GO_FILES)
	docker run \
		--rm \
		--name=$(PROJECT_NAME)-builder \
		-v $(PROJECT_NAME)-cache:/cache \
		-v $$(pwd):/app \
		-e GIT_TOKEN \
		-w /app \
		-e GOCACHE=/cache \
		-e LOCAL_USER=$$(id -u $$(logname)) \
		rsp/$(BUILDER_IMAGE):dev \
		./build.sh

build/openvino-runtime: Dockerfile.runtime
	docker build \
		$(PROXY_ARGS) \
		$(EXTRA_BUILD_ARGS) \
		--build-arg LOCAL_USER=$$(id -u $$(logname)) \
		-f Dockerfile.runtime \
		--label "git_sha=$(GIT_SHA)" \
		-t rsp/$(RUNTIME_IMAGE):dev \
		.
	@$(touch_target_file)

build/docker: build/openvino-runtime $(PROJECT_NAME) entrypoint.sh Dockerfile $(RES_FILES)
	docker build --rm \
		$(PROXY_ARGS) \
		$(EXTRA_BUILD_ARGS) \
		-f Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t rsp/$(PROJECT_NAME):dev \
		.
	@$(touch_target_file)


delete-all-recordings:
	sudo find recordings/ -mindepth 1 -delete

iterate: build up

iterate-d: build up-d
	$(trap_ctrl_c) $(MAKE) tail

restart:
	$(compose) restart $(args)

kill:
	$(compose) kill $(args)

tail:
	$(trap_ctrl_c) $(call log,-f --tail=10, $(args))

down:
	$(compose) down --remove-orphans $(args)

up: build
	xhost +
	$(compose) up --remove-orphans $(args)

up-d: build
	$(MAKE) up args="-d $(args)"

deploy: up-d

fmt:
	go fmt ./...

test:
	@echo "Go Testing..."
	docker run \
		--rm \
		--name=$(PROJECT_NAME)-tester \
		-v $(PROJECT_NAME)-cache:/cache \
		-v $$(pwd):/app \
		-e GIT_TOKEN \
		-w /app \
		-e GOCACHE=/cache \
		-e LOCAL_USER=$$(id -u $$(logname)) \
		rsp/$(BUILDER_IMAGE):dev \
		./unittests.sh ./... $(args)

force-test:
	$(MAKE) test args=-count=1

ps:
	$(compose) ps

shell/app:
	docker run -it --rm --entrypoint /bin/bash rsp/$(PROJECT_NAME):dev

shell/builder:
	docker run -it --rm --entrypoint /bin/bash rsp/$(BUILDER_IMAGE):dev

shell/runtime:
	docker run -it --rm --entrypoint /bin/bash rsp/$(RUNTIME_IMAGE):dev
