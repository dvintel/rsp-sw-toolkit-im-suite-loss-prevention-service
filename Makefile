# Apache v2 license
# Copyright (C) <2019> Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0
#

SERVICE_NAME ?= loss-prevention
PROJECT_NAME ?= loss-prevention-service

FULL_IMAGE_TAG ?= rsp/$(PROJECT_NAME):dev

POE_CAMERA ?= true
USB_CAMERA ?= 0
LIVE_VIEW ?= true

ifeq ($(POE_CAMERA),true)
SWARM_MODE = 1
endif

ifdef SWARM_MODE
FILE_FLAG = --compose-file
STACK_NAME = $(SERVICE_NAME)
log = docker logs $1 $2 $$(docker ps -qf name=$(STACK_NAME)_$(SERVICE_NAME).1) 2>&1
else
FILE_FLAG = -f
log = docker-compose logs $1 $2 2>&1
endif

COMPOSE_FILES = docker-compose.yml \
				$(if $(filter true,$(POE_CAMERA)),compose/poe-camera.yml,compose/usb-camera.yml) \
				$(if $(filter true,$(LIVE_VIEW)),compose/live-view.yml,)

ifndef GIT_COMMIT
GIT_COMMIT := $(shell git rev-parse HEAD)
endif

ifdef JENKINS_URL
GIT_TOKEN := $$(sed -nr 's|https://([^:]+):.+|\1|p' ~/.git-credentials)
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
TEST_ENV_VARS ?=

trap_ctrl_c = trap 'exit 0' INT;
compose = docker-compose

.PHONY: build clean test iterate tail stop deploy kill restart fmt ps delete-all-recordings shell

build: $(PROJECT_NAME)

build/docker: Dockerfile Makefile $(GO_FILES) $(RES_FILES) | build/
	docker build \
		$(PROXY_ARGS) \
		$(EXTRA_BUILD_ARGS) \
		--build-arg GIT_TOKEN=$(GIT_TOKEN) \
		-f Dockerfile \
		--label "git_commit=$(GIT_COMMIT)" \
		-t $(FULL_IMAGE_TAG) \
		.
	touch $@

$(PROJECT_NAME): build/docker
	container_id=$$(docker create $(FULL_IMAGE_TAG)) && \
		docker cp $${container_id}:/$(PROJECT_NAME) ./$(PROJECT_NAME) && \
		docker rm $${container_id}
	touch $@

clean:
	rm -rf build/*
	rm -f $(PROJECT_NAME)

delete-all-recordings:
	sudo find recordings/ -mindepth 1 -delete

iterate: build deploy

tail:
	$(trap_ctrl_c) $(call log,-f --tail=10, $(args))

ifdef SWARM_MODE
deploy: build
	xhost +
	USB_CAMERA=$(USB_CAMERA) \
		docker stack deploy \
		--with-registry-auth \
		$(addprefix $(FILE_FLAG) ,$(COMPOSE_FILES)) \
		$(args) \
		$(STACK_NAME)

stop:
	docker stack rm $(STACK_NAME) $(args)

ps:
	$(stack) ps $(STACK_NAME)

else

up: build
	xhost +
	USB_CAMERA=$(USB_CAMERA) \
		$(compose) \
		$(addprefix $(FILE_FLAG) ,$(COMPOSE_FILES)) \
		up \
		--remove-orphans \
		$(args)
	
deploy: build
	$(MAKE) up args="-d $(args)"

stop:
	$(compose) down --remove-orphans $(args)

ps:
	$(compose) ps

restart:
	$(compose) restart $(args)

kill:
	$(compose) kill $(args)

endif

fmt:
	go fmt ./...

# to prevent docker from caching the unit tests, a unique argument of the current unix epoch in nanoseconds in added (NANOSECONDS)
test:
	@echo "Go Testing..."
	docker build --rm \
		$(PROXY_ARGS) \
		$(EXTRA_BUILD_ARGS) \
		--build-arg GIT_TOKEN=$(GIT_TOKEN) \
		--build-arg TEST_COMMAND="$(TEST_ENV_VARS) NANOSECONDS=$$(date +%s%N) GOOS=linux GOARCH=amd64 CGO_ENABLED=1 GO111MODULE=auto go test ./... -v $(args)" \
		--target testing \
		-f Dockerfile \
		.

shell:
	docker run -it --rm --entrypoint /bin/bash rsp/$(PROJECT_NAME):dev

build/:
	@mkdir -p $@
