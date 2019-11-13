# Base image
FROM ubuntu:18.04 as builder-base
# Update package list, upgrade existing packages, and download dependencies
RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
        gstreamer1.0-plugins-base \
        libavcodec-dev \
        libavformat-dev \
        libcairo2-dev \
        libglib2.0-dev \
        libgstreamer1.0-0 \
        libgtk-3.0 \
        libgtk2.0-dev \
        libpango1.0-dev \
        libswscale-dev

RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
        build-essential \
        gnupg2 \
        libzmq3-dev \
        git \
        wget

FROM builder-base as openvino-builder
# Install OpenVINO
RUN wget https://apt.repos.intel.com/intel-gpg-keys/GPG-PUB-KEY-INTEL-SW-PRODUCTS-2019.PUB -O /tmp/key.pub && \
    apt-key add /tmp/key.pub && \
    echo "deb https://apt.repos.intel.com/openvino/2019/ all main" > /etc/apt/sources.list.d/intel-openvino-2019.list && \
    apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y intel-openvino-dev-ubuntu18-2019.3.344

FROM openvino-builder as gocv-openvino-builder
# Install Go 1.12
RUN wget "https://keyserver.ubuntu.com/pks/lookup?op=get&options=mr&search=0xF6BC817356A3D45E" -O /tmp/key.pub && \
    apt-key add /tmp/key.pub && \
    echo "deb http://ppa.launchpad.net/longsleep/golang-backports/ubuntu bionic main" > /etc/apt/sources.list.d/longsleep-ubuntu-golang-backports-bionic.list && \
    apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y golang-1.12-go
ENV PATH=/usr/lib/go-1.12/bin:$PATH

FROM gocv-openvino-builder as gocv-openvino-impcloud-builder
# Authentication needed to pull git modules from github.impcloud.net
ARG GIT_TOKEN
RUN git config --global credential.helper store && \
    set +x && \
    echo "https://$GIT_TOKEN:x-oauth-basic@github.impcloud.net" > ~/.git-credentials

FROM gocv-openvino-impcloud-builder as final-app-builder
# Download go modules first so they can be cached for faster subsequent builds
WORKDIR /app
COPY go.mod go.mod
ARG LOCAL_USER
RUN (go mod download && chown ${LOCAL_USER}:${LOCAL_USER} go.mod go.sum) || \
    (printf "\n\n\e[31mThere was an error downloading go module dependencies.\nPlease make sure you set \e[1mGIT_TOKEN\e[22;24m to your git auth token!\e[0m\n\n"; exit 1)
