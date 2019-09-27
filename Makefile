
SERVICE_NAME ?= loss-prevention
PROJECT_NAME ?= loss-prevention-service

BUILDER_IMAGE ?= rsp/$(PROJECT_NAME)-builder:dev

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

.PHONY: build build-builder loss-prevention-service docker

default: build

SRC_PATH := $(shell pwd | sed -nr 's|$(GOPATH)/src/(.+)|\1|p')
GO = GOOS=linux GOARCH=amd64 CGO_ENABLED=1 GO111MODULE=on go

builder:
	docker build --rm \
		--build-arg GIT_TOKEN=$(GIT_TOKEN) \
		--build-arg http_proxy=$(http_proxy) \
		--build-arg https_proxy=$(https_proxy) \
		-f Dockerfile.builder \
		--label "git_sha=$(GIT_SHA)" \
		-t $(BUILDER_IMAGE) \
		.

build: loss-prevention-service

loss-prevention-service:
	docker run \
		--rm \
		-it \
		--name=gobuilder \
		-v $(PROJECT_NAME)-cache:/cache \
		-v $(GOPATH)/src/$(SRC_PATH):/go/src/$(SRC_PATH) \
		-v ~/.git-credentials:/root/.git-credentials \
		-w /go/src/$(SRC_PATH) \
		-e GOCACHE=/cache \
		$(BUILDER_IMAGE) \
		sh -c '$(GO) build -v -o ./$@'

docker: build
	docker build --rm \
		--build-arg GIT_TOKEN=$(GIT_TOKEN) \
		--build-arg http_proxy=$(http_proxy) \
		--build-arg https_proxy=$(https_proxy) \
		-f Dockerfile_dev \
		--label "git_sha=$(GIT_SHA)" \
		-t rsp/$(PROJECT_NAME):dev \
		.

build-video:
	docker build --rm \
		--build-arg GIT_TOKEN=$(GIT_TOKEN) \
		--build-arg http_proxy=$(http_proxy) \
		--build-arg https_proxy=$(https_proxy) \
		-f video-server/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t rsp/video-server:dev \
		./video-server/

iterate:
	$(compose) down &
	$(MAKE) build docker
	# make sure it has stopped before we try and start it again
	$(call wait_for_service, stop, !)
	$(compose) up

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
	$(compose) up --remove-orphans $(args)

up: start
deploy: start

fmt:
	go fmt ./...

test:
	@$(call test,$(args))

force-test:
	@$(call test,-count=1)

ps:
	$(compose) ps

