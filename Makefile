
SERVICE_NAME ?= loss-prevention
PROJECT_NAME ?= loss-prevention-service

BUILDER_IMAGE ?= gocv-openvino-builder
RUNTIME_IMAGE ?= openvino-runtime

ifdef JENKINS_URL
JENKINS_CI_BUILD := 1
JENKINS_IMAGE_FULL_TAG ?= jenkins/$(PROJECT_NAME):dev
TEST_IMAGE_FULL_TAG = $(JENKINS_IMAGE_FULL_TAG)
BUILD_IMAGE_FULL_TAG = $(JENKINS_IMAGE_FULL_TAG)
APP_VOLUME_MOUNT :=
GIT_SHA = $(GIT_COMMIT)
NO_PROXY := localhost
no_proxy := localhost
LOCAL_USER := 0
GIT_TOKEN = $$(sed -nr 's|https://([^:]+):.+|\1|p' ~/.git-credentials)
EXTRA_TEST_REQS = build/jenkins-builder
else
TEST_IMAGE_FULL_TAG ?= rsp/$(BUILDER_IMAGE):dev
BUILD_IMAGE_FULL_TAG ?= rsp/$(BUILDER_IMAGE):dev
APP_VOLUME_MOUNT ?= -v $(PWD):/app
GIT_SHA := $(shell git rev-parse HEAD)
LOCAL_USER := $(shell id -u $$(logname))
endif

# The default flags to use when calling submakes
GNUMAKEFLAGS = --no-print-directory

GO_FILES := $(shell find . -type f -name '*.go')
RES_FILES := $(shell find res/ -type f)

PROXY_ARGS =	--build-arg http_proxy=$(http_proxy) \
				--build-arg https_proxy=$(https_proxy) \
				--build-arg no_proxy=$(no_proxy) \
				--build-arg HTTP_PROXY=$(HTTP_PROXY) \
				--build-arg HTTPS_PROXY=$(HTTPS_PROXY) \
				--build-arg NO_PROXY=$(NO_PROXY)

EXTRA_BUILD_ARGS ?=

trap_ctrl_c = trap 'exit 0' INT;
compose = docker-compose
log = docker-compose logs $1 $2 2>&1

.PHONY: build clean iterate iterate-d tail start stop rm deploy kill down fmt ps delete-all-recordings shell/*

build: build/docker

clean:
	rm -rf build/*
	rm -f $(PROJECT_NAME)

build/openvino-builder: go.mod Dockerfile.builder | build/
	docker build \
		--build-arg GIT_TOKEN=$(GIT_TOKEN) \
		$(PROXY_ARGS) \
		$(EXTRA_BUILD_ARGS) \
		--build-arg LOCAL_USER=$(LOCAL_USER) \
		-f Dockerfile.builder \
		--label "git_sha=$(GIT_SHA)" \
		-t rsp/$(BUILDER_IMAGE):dev \
		.
	touch $@

ifdef JENKINS_CI_BUILD
build/jenkins-builder: build/openvino-builder Dockerfile.jenkins Jenkinsfile build.sh $(GO_FILES) | build/
	docker build \
		-f Dockerfile.jenkins \
		-t $(JENKINS_IMAGE_FULL_TAG) \
		.
	touch $@

$(PROJECT_NAME): build/jenkins-builder | build
	container_id=$$(docker create $(JENKINS_IMAGE_FULL_TAG)) && \
		docker cp $${container_id}:/app/$(PROJECT_NAME) ./$(PROJECT_NAME) && \
		docker rm $${container_id}
	touch $@

else
$(PROJECT_NAME): build/openvino-builder Makefile build.sh $(GO_FILES)
	docker run \
		--rm \
		--name=$(PROJECT_NAME)-builder \
		-v $(PROJECT_NAME)-cache:/cache \
		$(APP_VOLUME_MOUNT) \
		$(EXTRA_BUILD_RUN_ARGS) \
		-e GIT_TOKEN \
		-w /app \
		-e GOCACHE=/cache \
		-e LOCAL_USER=$(LOCAL_USER) \
		$(BUILD_IMAGE_FULL_TAG) \
		./build.sh
endif

build/openvino-runtime: Dockerfile.runtime | build/
	docker build \
		$(PROXY_ARGS) \
		$(EXTRA_BUILD_ARGS) \
		--build-arg LOCAL_USER=$(LOCAL_USER) \
		-f Dockerfile.runtime \
		--label "git_sha=$(GIT_SHA)" \
		-t rsp/$(RUNTIME_IMAGE):dev \
		.
	touch $@

build/docker: build/openvino-runtime $(PROJECT_NAME) entrypoint.sh Dockerfile $(RES_FILES) | build/
	docker build --rm \
		$(PROXY_ARGS) \
		$(EXTRA_BUILD_ARGS) \
		-f Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t rsp/$(PROJECT_NAME):dev \
		.
	touch $@


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

test: unittests.sh $(EXTRA_TEST_REQS)
	@echo "Go Testing..."
	docker run \
		--rm \
		--name=$(PROJECT_NAME)-tester \
		-v $(PROJECT_NAME)-cache:/cache \
		$(APP_VOLUME_MOUNT) \
		-e GIT_TOKEN \
		-w /app \
		-e GOCACHE=/cache \
		-e LOCAL_USER=$(LOCAL_USER) \
		$(TEST_IMAGE_FULL_TAG) \
		./unittests.sh ./... $(args)

force-test:
	$(MAKE) test args="-count=1"

ps:
	$(compose) ps

shell/app:
	docker run -it --rm --entrypoint /bin/bash rsp/$(PROJECT_NAME):dev

shell/builder:
	docker run -it --rm --entrypoint /bin/bash rsp/$(BUILDER_IMAGE):dev

shell/runtime:
	docker run -it --rm --entrypoint /bin/bash rsp/$(RUNTIME_IMAGE):dev

build/:
	@mkdir -p $@

