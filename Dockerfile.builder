# Base image
FROM ubuntu:18.04 as builder-base
# Update package list and upgrade existing packages
RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get upgrade -y
# Download dependencies
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y \
        build-essential \
        cmake \
        git \
        gnupg2 \
        gstreamer1.0-plugins-base \
        libavcodec-dev \
        libavformat-dev \
        libcairo2-dev \
        libglib2.0-dev \
        libgstreamer1.0-0 \
        libgtk-3.0 \
        libgtk2.0-dev \
        libpango1.0-dev \
        libswscale-dev \
        libzmq3-dev \
        python3 \
        python3-pip \
        software-properties-common \
        sudo \
        vim \
        virtualenv \
        wget

FROM builder-base as openvino-builder
# Install OpenVINO
RUN wget https://apt.repos.intel.com/intel-gpg-keys/GPG-PUB-KEY-INTEL-SW-PRODUCTS-2019.PUB -O /tmp/key.pub && \
    apt-key add /tmp/key.pub && \
    echo "deb https://apt.repos.intel.com/openvino/2019/ all main" > /etc/apt/sources.list.d/intel-openvino-2019.list && \
    apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y intel-openvino-dev-ubuntu18-2019.3.344 && \
    pip3 install -r /opt/intel/openvino/python/python3.6/requirements.txt && \
    cd /opt/intel/openvino/deployment_tools/inference_engine/samples && \
    ./build_samples.sh

FROM openvino-builder as gocv-openvino-builder
# Install Go 1.12
RUN add-apt-repository ppa:longsleep/golang-backports -y && \
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

FROM final-app-builder
# Do nothing
