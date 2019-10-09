FROM golang:1.12-alpine as go112

FROM denismakogon/gocv-alpine:4.0.1-buildstage as gobuilder

RUN echo http://nl.alpinelinux.org/alpine/v3.6/main > /etc/apk/repositories; \
    echo http://nl.alpinelinux.org/alpine/v3.6/community >> /etc/apk/repositories

RUN apk update && apk add zeromq zeromq-dev libsodium-dev pkgconfig build-base bash util-linux

RUN apk update && \
    apk add --no-cache linux-headers gcc musl-dev git && \
    apk add --no-cache \
        --virtual=.build-dependencies &&\
    ln -s locale.h /usr/include/xlocale.h && \
    rm /usr/include/xlocale.h

# Authentication needed to pull git modules from github.impcloud.net
RUN git config --global credential.helper store
ARG GIT_TOKEN
RUN set +x && echo "https://$GIT_TOKEN:x-oauth-basic@github.impcloud.net" > ~/.git-credentials

# Upgrade the version of go
RUN rm -rf /go /usr/local/go
COPY --from=go112 /go /go
COPY --from=go112 /usr/local/go /usr/local/go

WORKDIR /app
# Download go modules first so they can be cached for faster subsequent builds
COPY go.mod go.mod
RUN go mod download && mv go.sum go.sum.bak
