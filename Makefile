SERVICE_NAME ?= loss-prevention
PROJECT_NAME ?= loss-prevention-service

FULL_IMAGE_TAG ?= rsp/$(PROJECT_NAME):dev

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

trap_ctrl_c = trap 'exit 0' INT;
compose = docker-compose
log = docker-compose logs $1 $2 2>&1

.PHONY: build clean test force-test iterate iterate-d tail start stop rm deploy kill down fmt ps delete-all-recordings shell

build: build/docker

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

clean:
	rm -rf build/*

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

test: build/docker
	@echo "Go Testing..."
	docker run \
		--rm \
		--name=$(PROJECT_NAME)-tester \
		-w /app \
		--entrypoint "/usr/lib/go-1.12/bin/go" \
		-e GOOS=linux \
		-e GOARCH=amd64 \
		-e CGO_ENABLED=1 \
		-e GO111MODULE=auto \
		$(FULL_IMAGE_TAG) \
		test -v $(args) ./...

force-test:
	$(MAKE) test args="-count=1"

ps:
	$(compose) ps

shell:
	docker run -it --rm --entrypoint /bin/bash rsp/$(PROJECT_NAME):dev

build/:
	@mkdir -p $@
