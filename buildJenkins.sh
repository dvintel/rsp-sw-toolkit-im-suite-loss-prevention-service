#!/bin/bash

set -e

binary_name=loss-prevention-service
jenkinsImageName=jenkins/${binary_name}:dev
GIT_TOKEN=$(sed -nr 's|https://([^:]+):.+|\1|p' ~/.git-credentials)

make build-jenkins TAG=${jenkinsImageName} no_proxy="" NO_PROXY="" LOCAL_USER=0 GIT_TOKEN=${GIT_TOKEN} APP_VOLUME_MOUNT=""

container_id=$(docker create ${jenkinsImageName})
docker cp ${container_id}:/app/${binary_name} ./${binary_name}
docker rm ${container_id}

make build no_proxy="" NO_PROXY="" LOCAL_USER=0 GIT_TOKEN=${GIT_TOKEN} APP_VOLUME_MOUNT=""

make force-test TEST_IMAGE_FULL_TAG=${jenkinsImageName} LOCAL_USER=0 GIT_TOKEN=${GIT_TOKEN} APP_VOLUME_MOUNT=""
