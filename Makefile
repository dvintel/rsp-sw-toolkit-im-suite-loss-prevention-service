
SERVICE_NAME ?= loss-prevention
PROJECT_NAME ?= loss-prevention-service

#BUILDER_IMAGE ?= rsp/$(PROJECT_NAME)-builder:dev
BUILDER_IMAGE ?= rsp/openvino:dev

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

.PHONY: build build-builder loss-prevention-service docker

default: build

build: vino Makefile build.sh
	docker run \
		--rm \
		-it \
		--name=gobuilder \
		-v $(PROJECT_NAME)-cache:/cache \
		-v $$(pwd):/app \
		-e GIT_TOKEN \
		-w /app \
		-e GOCACHE=/cache \
		-e LOCAL_USER=$$(id -u $$(logname)) \
		$(BUILDER_IMAGE) \
		bash -c '/app/build.sh'

docker: build Dockerfile
	docker build --rm \
		--build-arg GIT_TOKEN=$(GIT_TOKEN) \
		--build-arg http_proxy=$(http_proxy) \
		--build-arg https_proxy=$(https_proxy) \
		-f Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t rsp/$(PROJECT_NAME):dev \
		.

vino: go.mod Dockerfile.vino
	docker build \
		--build-arg GIT_TOKEN=$(GIT_TOKEN) \
		--build-arg http_proxy=$(http_proxy) \
		--build-arg https_proxy=$(https_proxy) \
		--build-arg LOCAL_USER=$$(id -u $$(logname)) \
		-f Dockerfile.vino \
		--label "git_sha=$(GIT_SHA)" \
		-t rsp/openvino:dev \
		.

iterate:
	$(compose) down --remove-orphans &
	$(MAKE) build docker
	# make sure it has stopped before we try and start it again
	$(call wait_for_service, stop, !)
	$(compose) up --remove-orphans

iterate-d:
	$(compose) down --remove-orphans &
	$(MAKE) build docker
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

