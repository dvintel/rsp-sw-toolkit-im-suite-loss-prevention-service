#!/usr/bin/env bash

. /opt/intel/openvino/bin/setupvars.sh
cd /
exec /loss-prevention-service -r --profile=docker --confdir=/res
