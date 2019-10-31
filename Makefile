
SERVICE_NAME ?= loss-prevention
PROJECT_NAME ?= loss-prevention-service

BUILDER_IMAGE ?= gocv-openvino-builder
RUNTIME_IMAGE ?= openvino-runtime

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

trap_ctrl_c = trap 'exit 0' INT;

compose = docker-compose

get_id = `docker ps -qf name=$(SERVICE_NAME)_1`

wait_for_service =	@printf "Waiting for $(SERVICE_NAME) service to$1..."; \
					while [  $2 -z $(get_id) ]; \
                 	do \
                 		printf "."; \
                 		sleep 0.3;\
                 	done; \
                 	printf "\n";

log = docker-compose logs $1 $2 2>&1

test =	echo "Go Testing..."; \
		go test ./... $1

.PHONY: build iterate iterate-d tail start stop rm deploy kill down fmt ps app-shell builder-shell runtime-shell delete-all-recordings

default: build

build: build/docker

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
	@mkdir -p $(@D) && touch $@

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
		bash -c '/app/build.sh'

build/openvino-runtime: Dockerfile.runtime
	docker build \
		$(PROXY_ARGS) \
		$(EXTRA_BUILD_ARGS) \
		--build-arg LOCAL_USER=$$(id -u $$(logname)) \
		-f Dockerfile.runtime \
		--label "git_sha=$(GIT_SHA)" \
		-t rsp/$(RUNTIME_IMAGE):dev \
		.
	@mkdir -p $(@D) && touch $@

build/docker: build/openvino-runtime $(PROJECT_NAME) entrypoint.sh Dockerfile $(RES_FILES)
	docker build --rm \
		$(PROXY_ARGS) \
		$(EXTRA_BUILD_ARGS) \
		-f Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t rsp/$(PROJECT_NAME):dev \
		.
	@mkdir -p $(@D) && touch $@

# Helper target which builds the builder and runtime containers in separate processes to speed up build times
# Note that the & is not a typo. It is singular in order to fork the process, as opposed to && which means 'and'
# Press Ctrl-C to kill
prepare:
	@$(MAKE) --no-print-directory build/openvino-builder & \
	$(MAKE) --no-print-directory build/openvino-runtime & \
	trap "kill -- -$$(ps -o pgid= $$$$ | tr -d ' ')" INT; wait


delete-all-recordings:
	sudo find recordings/ -mindepth 1 -delete

iterate:
	$(compose) down --remove-orphans &
	$(MAKE) build
	$(call wait_for_service, stop, !)
	$(compose) up --remove-orphans

iterate-d:
	$(compose) down --remove-orphans &
	$(MAKE) build
	$(call wait_for_service, stop, !)
	$(compose) up --remove-orphans -d
	$(MAKE) tail

restart:
	$(compose) restart $(args)

kill:
	$(compose) kill $(args)

tail:
	$(trap_ctrl_c) $(call log,-f --tail=10, $(args))

rm:
	$(compose) rm --remove-orphans $(args)

down:
	$(compose) down --remove-orphans $(args)

up:
	$(compose) up --remove-orphans $(args)

stop: down
start: up

up-d:
	$(MAKE) up args="-d $(args)"

deploy: up-d

fmt:
	go fmt ./...

test:
	@$(call test,$(args))

force-test:
	@$(call test,-count=1)

ps:
	$(compose) ps

app-shell:
	docker run -it --rm --entrypoint /bin/bash rsp/$(PROJECT_NAME):dev

builder-shell:
	docker run -it --rm --entrypoint /bin/bash rsp/$(BUILDER_IMAGE):dev

runtime-shell:
	docker run -it --rm --entrypoint /bin/bash rsp/$(RUNTIME_IMAGE):dev
