#!/bin/bash

set -e

# NOTE: This script is only intended to be run within the builder docker container, NOT natively

app_name=loss-prevention-service

fixPermissions() {
  printf "\e[36mFixing permissions...\e[0m"
  chown ${LOCAL_USER}:${LOCAL_USER} go.mod go.sum
  printf "\e[2m\e[32m[OK]\e[0m\n"
}

printf "\e[34mBuilding ${app_name}...\e[0m\n"
GOOS=linux GOARCH=amd64 CGO_ENABLED=1 GO111MODULE=auto go build -v -o ./${app_name} || (err=$?; fixPermissions; exit ${err})
printf "\e[34mBuild finished \e[32m[OK]\e[0m\n"

fixPermissions
