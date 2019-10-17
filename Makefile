
SERVICE_NAME ?= loss-prevention
PROJECT_NAME ?= loss-prevention-service

BUILDER_IMAGE ?= gocv-openvino-builder
RUNTIME_IMAGE ?= openvino-runtime

GIT_SHA = $(shell git rev-parse HEAD)

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

.PHONY: build compile docker iterate iterate-d tail start stop rm deploy kill down fmt ps shell

default: build

build: compile docker

openvino-builder: go.mod Dockerfile.builder
	docker build \
		--build-arg GIT_TOKEN=$(GIT_TOKEN) \
		--build-arg http_proxy=$(http_proxy) \
		--build-arg https_proxy=$(https_proxy) \
		--build-arg LOCAL_USER=$$(id -u $$(logname)) \
		-f Dockerfile.builder \
		--label "git_sha=$(GIT_SHA)" \
		-t rsp/$(BUILDER_IMAGE):dev \
		.

openvino-runtime: Dockerfile.runtime
	docker build \
		--build-arg http_proxy=$(http_proxy) \
		--build-arg https_proxy=$(https_proxy) \
		--build-arg LOCAL_USER=$$(id -u $$(logname)) \
		-f Dockerfile.runtime \
		--label "git_sha=$(GIT_SHA)" \
		-t rsp/$(RUNTIME_IMAGE):dev \
		.

compile: openvino-builder Makefile build.sh
	docker run \
		--rm \
		-it \
		--name=$(PROJECT_NAME)-builder \
		-v $(PROJECT_NAME)-cache:/cache \
		-v $$(pwd):/app \
		-e GIT_TOKEN \
		-w /app \
		-e GOCACHE=/cache \
		-e LOCAL_USER=$$(id -u $$(logname)) \
		rsp/$(BUILDER_IMAGE):dev \
		bash -c '/app/build.sh'

docker: compile openvino-runtime Dockerfile
	docker build --rm \
		--build-arg GIT_TOKEN=$(GIT_TOKEN) \
		--build-arg http_proxy=$(http_proxy) \
		--build-arg https_proxy=$(https_proxy) \
		-f Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t rsp/$(PROJECT_NAME):dev \
		.

delete-all-recordings:
	sudo find recordings/ -type f -delete

iterate:
	$(compose) down --remove-orphans &
	$(MAKE) build
	# make sure it has stopped before we try and start it again
	$(call wait_for_service, stop, !)
	$(compose) up --remove-orphans

iterate-d:
	$(compose) down --remove-orphans &
	$(MAKE) build
	# make sure it has stopped before we try and start it again
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
