# Apache v2 license
# Copyright (C) <2019> Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0
#

# Base image
FROM ubuntu:18.04 as builder-base
# Install OS dependencies
RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get upgrade -y && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
        gnupg2 \
        libzmq3-dev \
        git \
        wget && \
    DEBIAN_FRONTEND=noninteractive apt-get autoremove -y

FROM builder-base as gocv-builder
# Install Go 1.12
RUN wget "https://keyserver.ubuntu.com/pks/lookup?op=get&options=mr&search=0xF6BC817356A3D45E" -O /tmp/key.pub && \
    apt-key add /tmp/key.pub && \
    echo "deb http://ppa.launchpad.net/longsleep/golang-backports/ubuntu bionic main" > /etc/apt/sources.list.d/longsleep-ubuntu-golang-backports-bionic.list && \
    apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y golang-1.12-go
ENV PATH=/usr/lib/go-1.12/bin:$PATH

WORKDIR /tmp
# Install gocv
# Note: gocv install script calls `sudo` command which is not available. However, because docker is running the build
#       as root anyways, we can just create a wrapper script in its place instead of actually installing or using it
RUN printf '#!/bin/sh\nexec "$@"\n' > /bin/sudo && \
    chmod +x /bin/sudo && \
    git clone -b v0.21.0 https://github.com/hybridgroup/gocv && \
    make -C ./gocv deps download build sudo_install verify && \
    rm -f /bin/sudo && \
    cp -a /tmp/opencv/opencv-*/data /data && \
    chown -R 2000:2000 /data

FROM gocv-builder as app-builder
# Download go modules first so they can be cached for faster subsequent builds
WORKDIR /app
COPY go.mod go.mod
RUN go mod download
# Pre-compile gocv lib to make subsequent builds faster
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=1 GO111MODULE=auto go install -v -x gocv.io/x/gocv 2>&1

FROM app-builder as app
# Copy the app code and build it
WORKDIR /app
COPY . .
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=1 GO111MODULE=auto go build -v -o ./loss-prevention-service

FROM app as testing
ARG TEST_COMMAND="echo 'Skipping unit tests.'"
RUN bash -c "set -x; $TEST_COMMAND; set +x;"

FROM app as final
WORKDIR /
COPY --chown=2000:2000 /res /res
COPY --chown=2000:2000 --from=app /app/loss-prevention-service /
RUN rm -rf /app

ENTRYPOINT ["/loss-prevention-service"]
CMD ["-r", "--profile=docker", "--confdir=/res"]
