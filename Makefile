
SERVICE_NAME ?= loss-prevention
PROJECT_NAME ?= loss-prevention-service

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

log = docker-compose logs $1$2 2>&1

test =	echo "Go Testing..."; \
		go test ./... $1

.PHONY: build

default: build

build:
	docker build --rm \
		--build-arg GIT_TOKEN=$(GIT_TOKEN) \
		--build-arg http_proxy=$(http_proxy) \
		--build-arg https_proxy=$(https_proxy) \
		-f Dockerfile_dev \
		--label "git_sha=$(GIT_SHA)" \
		-t rsp/$(PROJECT_NAME):dev \
		.

iterate:
	$(compose) down &
	$(MAKE) build
	# make sure it has stopped before we try and start it again
	$(call wait_for_service, stop, !)
	$(compose) up -d
	$(call wait_for_service, start)
	$(MAKE) tail

restart:
	$(compose) down
	$(call wait_for_service, stop, !)
	$(compose) up -d
	$(call wait_for_service, start)

tail:
	$(trap_ctrl_c) $(call log,-f --tail 10,$(args))

stop:
	$(compose) down --remove-orphans

down: stop

start:
	$(compose) up -d $(args)

up: start

fmt:
	go fmt ./...

test:
	@$(call test,$(args))

force-test:
	@$(call test,-count=1)

ps:
	$(compose) ps

