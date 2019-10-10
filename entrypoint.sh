#!/usr/bin/env bash

set -e

printf "\e[36mSetting up OpenVINO environment...\e[0m"
. /opt/intel/openvino/bin/setupvars.sh > /dev/null
printf "\e[32m [OK]\e[0m\n"

exec /loss-prevention-service -r --profile=docker --confdir=/res
